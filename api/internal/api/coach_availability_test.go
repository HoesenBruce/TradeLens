package api_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/config"
	"github.com/tradermemos/api/internal/ocr"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestCoachIsolationAcrossConfigurationAndRestart(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "coach-owner", "ISOA", 10)
	member := f.seed(t, "coach-member", "ISOB", 20)
	q := store.NewForDriver(f.conn, "sqlite")
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); fmt.Fprint(w, "SYNTHETIC-PRIVATE") }))
	defer provider.Close()
	// Deliberately seed malformed JSON as well as valid secret-bearing legacy content.
	for _, u := range []isolationUser{owner, member} {
		for _, notes := range []string{`[{"headline":"SYNTHETIC-PRIVATE","detail":"SYNTHETIC-PRIVATE"}]`, `SYNTHETIC-PRIVATE invalid JSON`} {
			_, err := q.CreateCoachReview(context.Background(), store.CreateCoachReviewParams{ID: uuid.NewString(), UserID: u.id, TradeID: u.trade, Model: "SYNTHETIC-PRIVATE", Notes: notes, NextAction: "SYNTHETIC-PRIVATE"})
			require.NoError(t, err)
		}
	}
	for _, u := range []isolationUser{owner, member} {
		for _, route := range []string{"/exports", "/exports/trades"} {
			for _, format := range []string{"json", "csv", "zip"} {
				rec := do(f.s, "GET", "/api/v1"+route+"?format="+format+"&account_id="+u.accounts[0], "", u.session)
				require.Equal(t, 200, rec.Code)
				if format == "zip" {
					archive, err := zip.NewReader(bytes.NewReader(rec.Body.Bytes()), int64(rec.Body.Len()))
					require.NoError(t, err)
					for _, file := range archive.File {
						reader, err := file.Open()
						require.NoError(t, err)
						data, err := io.ReadAll(reader)
						require.NoError(t, err)
						require.NoError(t, reader.Close())
						isolationContains(t, string(data), "SYNTHETIC-PRIVATE", false)
					}
				} else {
					isolationContains(t, rec.Body.String(), "SYNTHETIC-PRIVATE", false)
				}
			}
		}
		for _, route := range []string{"/news/export", "/news/" + u.news + "/export", "/public/share/" + u.shareToken} {
			rec := do(f.s, "GET", "/api/v1"+route, "", u.session)
			require.Equal(t, 200, rec.Code)
			isolationContains(t, rec.Body.String(), "SYNTHETIC-PRIVATE", false)
		}
	}
	// Rebuild against the same database to exercise persisted upgraded settings and stale JWT/PATs.
	for _, enabled := range []bool{false, true} {
		jwt := auth.NewJWT("disposable-isolation-test-secret")
		server := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), CoachDefaults: ocr.VisionConfig{Enabled: true, BaseURL: provider.URL, APIKey: "SYNTHETIC-PRIVATE", Model: "SYNTHETIC-PRIVATE", CustomPrompt: "SYNTHETIC-PRIVATE"}})
		_, err := q.UpsertCoachSettings(context.Background(), store.UpsertCoachSettingsParams{Enabled: map[bool]int64{false: 0, true: 1}[enabled], BaseUrl: provider.URL, ApiKey: "SYNTHETIC-PRIVATE", Model: "SYNTHETIC-PRIVATE", CustomPrompt: "SYNTHETIC-PRIVATE"})
		require.NoError(t, err)
		before := f.databaseState(t)
		for _, u := range []isolationUser{owner, member} {
			for _, token := range []string{u.session, u.pat} {
				capability := do(server, "GET", "/api/v1/capabilities/coach", "", token)
				require.Equal(t, 200, capability.Code)
				require.Equal(t, "no-store", capability.Header().Get("Cache-Control"))
				require.JSONEq(t, `{"available":false,"reason":"unavailable","news_analysis_available":false}`, capability.Body.String())
				for _, suffix := range []string{"/coach", "/coach/stream", "/coach?output_version=2", "/coach/stream?output_version=2"} {
					rec := do(server, "POST", "/api/v1/trades/"+u.trade+suffix, `{}`, token)
					require.Equal(t, 200, rec.Code)
					require.Equal(t, "no-store", rec.Header().Get("Cache-Control"))
					isolationContains(t, rec.Body.String(), `"source":"off"`, true)
					isolationContains(t, rec.Body.String(), "event: note", false)
					isolationContains(t, rec.Body.String(), "SYNTHETIC-PRIVATE", false)
				}
				history := do(server, "GET", "/api/v1/trades/"+u.trade+"/coach/reviews", "", token)
				require.Equal(t, 200, history.Code)
				require.Equal(t, "no-store", history.Header().Get("Cache-Control"))
				var got struct {
					Reviews []struct {
						Status            string
						Notes             []any
						Model, NextAction string
					}
				}
				require.NoError(t, json.Unmarshal(history.Body.Bytes(), &got))
				require.Len(t, got.Reviews, 2)
				for _, r := range got.Reviews {
					require.Equal(t, "legacy_withheld", r.Status)
					require.Empty(t, r.Notes)
					require.Empty(t, r.Model)
					require.Empty(t, r.NextAction)
				}
				isolationContains(t, history.Body.String(), "SYNTHETIC-PRIVATE", false)
				require.Equal(t, 503, do(server, "POST", "/api/v1/news/"+u.news+"/analyze", `{}`, token).Code)
				other := owner
				if u.id == owner.id {
					other = member
				}
				for _, suffix := range []string{"/coach", "/coach/stream"} {
					require.Equal(t, 404, do(server, "POST", "/api/v1/trades/"+other.trade+suffix, `{}`, token).Code)
				}
				require.Equal(t, 404, do(server, "GET", "/api/v1/trades/"+other.trade+"/coach/reviews", "", token).Code)
				require.Equal(t, 404, do(server, "POST", "/api/v1/news/"+other.news+"/analyze", `{}`, token).Code)
			}
		}
		for _, token := range []string{owner.session, owner.pat} {
			for _, suffix := range []string{"/test", "/models"} {
				require.Equal(t, 503, do(server, "POST", "/api/v1/settings/coach"+suffix, `{}`, token).Code)
			}
		}
		for _, path := range []string{"/capabilities/coach", "/trades/" + owner.trade + "/coach/reviews"} {
			require.Equal(t, 401, do(server, "GET", "/api/v1"+path, "", "").Code)
		}
		for _, path := range []string{"/trades/" + owner.trade + "/coach", "/trades/" + owner.trade + "/coach/stream", "/news/" + owner.news + "/analyze"} {
			require.Equal(t, 401, do(server, "POST", "/api/v1"+path, `{}`, "").Code)
		}
		var wg sync.WaitGroup
		for i := 0; i < 8; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				require.Equal(t, 200, do(server, "POST", "/api/v1/trades/"+owner.trade+"/coach/stream", `{}`, owner.session).Code)
			}()
		}
		wg.Wait()
		require.Zero(t, calls.Load())
		require.Equal(t, before, f.databaseState(t))
	}
}

func TestCoachNewInstallationAndEnvironmentCannotEnableLegacy(t *testing.T) {
	for _, enabled := range []string{"false", "true"} {
		t.Run(enabled, func(t *testing.T) {
			t.Setenv("TM_COACH_ENABLED", enabled)
			t.Setenv("TM_COACH_BASE_URL", "http://127.0.0.1:1")
			t.Setenv("TM_COACH_API_KEY", "synthetic-unused")
			cfg, err := config.Load()
			require.NoError(t, err)
			require.Equal(t, enabled == "true", cfg.CoachEnabled)
			f := newIsolationFixture(t)
			q := store.NewForDriver(f.conn, "sqlite")
			_, err = q.GetCoachSettings(context.Background())
			require.Error(t, err)
			jwt := auth.NewJWT("disposable-isolation-test-secret")
			server := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), CoachDefaults: ocr.VisionConfig{Enabled: cfg.CoachEnabled, BaseURL: cfg.CoachBaseURL, APIKey: cfg.CoachAPIKey, Model: cfg.CoachModel}})
			token := registerAndLogin(t, server, "fresh-"+enabled+"@synthetic.invalid")
			account := accountID(t, server, token)
			trade := closedTradeID(t, server, token, account)
			before := f.databaseState(t)
			require.JSONEq(t, `{"available":false,"reason":"unavailable","news_analysis_available":false}`, do(server, "GET", "/api/v1/capabilities/coach", "", token).Body.String())
			for _, suffix := range []string{"/coach", "/coach/stream"} {
				rec := do(server, "POST", "/api/v1/trades/"+trade+suffix, `{}`, token)
				require.Equal(t, 200, rec.Code)
				isolationContains(t, rec.Body.String(), `"source":"off"`, true)
			}
			require.Equal(t, before, f.databaseState(t))
		})
	}
}
