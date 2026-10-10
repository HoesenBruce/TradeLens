package api_test

import (
	"context"
	"encoding/json"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/alerts"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestReviewAcceptanceIsolationAndAlertPersistence(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "review.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	require.NoError(t, db.Migrate(conn))
	q := store.NewForDriver(conn, "sqlite")
	j := auth.NewJWT("test")
	s := api.New(api.Deps{JWT: j, Auth: auth.NewService(q, j, true), Store: q, Trades: trades.NewService(q)})
	a := registerAndLogin(t, s, "review-a@example.com")
	b := registerAndLogin(t, s, "review-b@example.com")
	accA, accB := accountID(t, s, a), accountID(t, s, b)
	idA, idB := closedTradeID(t, s, a, accA), closedTradeID(t, s, b, accB)
	read := func(id, token string) map[string]any {
		r := do(s, http.MethodGet, "/api/v1/trades/"+id, "", token)
		require.Equal(t, http.StatusOK, r.Code, r.Body.String())
		var value map[string]any
		require.NoError(t, json.Unmarshal(r.Body.Bytes(), &value))
		return value
	}
	before := read(idB, b)
	patch := `{"trade_quality":5,"notes":"forged review","tag_ids":[]}`
	for _, method := range []string{http.MethodGet, http.MethodPatch} {
		require.Equal(t, http.StatusNotFound, do(s, method, "/api/v1/trades/"+idB, patch, a).Code)
		require.Equal(t, http.StatusUnauthorized, do(s, method, "/api/v1/trades/"+idB, patch, "").Code)
	}
	for _, ids := range []string{accB, accA + "," + accB} {
		r := do(s, http.MethodGet, "/api/v1/trades?status=closed&account_id="+ids, "", a)
		// A forged filter may be rejected or safely filtered, but never expose B.
		if r.Code == http.StatusOK {
			require.NotContains(t, r.Body.String(), idB)
		} else {
			require.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusForbidden}, r.Code)
		}
	}
	require.Equal(t, http.StatusForbidden, adminCall(t, s, http.MethodGet, "/admin/backup", "", b).Code)
	require.Equal(t, http.StatusUnauthorized, prefsCall(t, s, http.MethodPatch, `{}`, "").Code)
	require.Equal(t, before, read(idB, b))
	var journals int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM trade_journal WHERE trade_id = ?", idB).Scan(&journals))
	require.Zero(t, journals, "unauthorized review must not persist a journal")

	var userA, userB string
	require.NoError(t, conn.QueryRow("SELECT user_id FROM accounts WHERE id = ?", accA).Scan(&userA))
	require.NoError(t, conn.QueryRow("SELECT user_id FROM accounts WHERE id = ?", accB).Scan(&userB))
	ctx := context.Background()
	for _, uid := range []string{userA, userB} {
		_, err = q.UpsertAlertSettings(ctx, store.UpsertAlertSettingsParams{UserID: uid, Enabled: 1, Timezone: "UTC", RuleUnreviewed: 1, UnreviewedDays: 7})
		require.NoError(t, err)
	}
	require.Equal(t, http.StatusOK, do(s, http.MethodPatch, "/api/v1/trades/"+idA, `{"notes":"notes alone"}`, a).Code)
	service := alerts.NewService(q, nil, false)
	require.NoError(t, service.EvaluateUser(ctx, userA))
	events, err := q.ListAlertEvents(ctx, store.ListAlertEventsParams{UserID: userA, Limit: 100})
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Contains(t, events[0].Body, "1 closed trade")
	// A forged B account key remains in A's opaque preferences only.
	cutoff := time.Now().UTC().Format(time.RFC3339Nano)
	require.Equal(t, http.StatusOK, prefsCall(t, s, http.MethodPatch, `{"reviewBacklogCutoff:`+accA+`":"`+cutoff+`","reviewBacklogCutoff:`+accB+`":"`+cutoff+`"}`, a).Code)
	require.Empty(t, readPrefs(t, prefsCall(t, s, http.MethodGet, "", b)).Prefs)
	stored, err := q.GetUserPreferences(ctx, userA)
	require.NoError(t, err)
	rows, err := q.ListClosedTrades(ctx, store.ListClosedTradesParams{UserID: userA})
	require.NoError(t, err)
	require.Zero(t, alerts.CountUnreviewed(rows, nil, time.Now(), alerts.ReviewCutoffs(stored.Prefs)))
	require.NoError(t, service.EvaluateUser(ctx, userB))
	otherEvents, err := q.ListAlertEvents(ctx, store.ListAlertEventsParams{UserID: userB, Limit: 100})
	require.NoError(t, err)
	require.Len(t, otherEvents, 1, "A's cutoff cannot suppress B's alert")
	require.Equal(t, http.StatusOK, prefsCall(t, s, http.MethodPatch, `{"reviewBacklogCutoff:`+accA+`":null}`, a).Code)
	require.NoError(t, service.EvaluateUser(ctx, userA))
	afterEvents, err := q.ListAlertEvents(ctx, store.ListAlertEventsParams{UserID: userA, Limit: 100})
	require.NoError(t, err)
	require.Equal(t, events, afterEvents, "weekly dedupe keeps delivered events unchanged")
	require.Equal(t, before, read(idB, b))
}
