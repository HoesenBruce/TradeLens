package alerts

import (
	"database/sql"
	"github.com/tradermemos/api/internal/store"
	"testing"
	"time"
)

func TestReviewCountGradeCutoffAndAccounts(t *testing.T) {
	now := time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)
	closed := func(days int) sql.NullTime { return sql.NullTime{Time: now.AddDate(0, 0, -days), Valid: true} }
	rows := []store.Trade{{ID: "notes", AccountID: "a", ClosedAt: closed(20)}, {ID: "other", AccountID: "b", ClosedAt: closed(20)}, {ID: "graded", AccountID: "b", ClosedAt: closed(10)}, {ID: "recent", AccountID: "a", ClosedAt: closed(1)}, {ID: "open", AccountID: "a"}}
	journals := map[string]store.TradeJournal{"notes": {Notes: "a lesson alone"}, "graded": {TradeQuality: sql.NullInt64{Int64: 4, Valid: true}}}
	if got := CountUnreviewed(rows, journals, now.AddDate(0, 0, -7), nil); got != 2 {
		t.Fatalf("notes only remain unreviewed: %d", got)
	}
	prefs := `{"reviewBacklogCutoff:a":"2026-09-26T12:00:00Z","reviewBacklogCutoff:b":null,"reviewBacklogCutoff:bad":"invalid"}`
	if got := CountUnreviewed(rows, journals, now.AddDate(0, 0, -7), ReviewCutoffs(prefs)); got != 1 {
		t.Fatalf("dismissal must only affect a: %d", got)
	}
	for _, n := range []int64{0, 1, 5, 6} {
		if Graded(sql.NullInt64{Int64: n, Valid: true}) != (n >= 1 && n <= 5) {
			t.Fatal(n)
		}
	}
	if Graded(sql.NullInt64{}) {
		t.Fatal("missing grade")
	}
}
