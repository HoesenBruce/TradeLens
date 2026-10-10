package alerts

import (
	"database/sql"
	"encoding/json"
	"github.com/tradermemos/api/internal/store"
	"time"
)

func Graded(quality sql.NullInt64) bool {
	return quality.Valid && quality.Int64 >= 1 && quality.Int64 <= 5
}

// Cutoffs are per brokerage account and scoped to the authenticated user by preferences.
func ReviewCutoffs(raw string) map[string]time.Time {
	var prefs map[string]json.RawMessage
	_ = json.Unmarshal([]byte(raw), &prefs)
	cutoffs := map[string]time.Time{}
	const prefix = "reviewBacklogCutoff:"
	for key, value := range prefs {
		if len(key) <= len(prefix) || key[:len(prefix)] != prefix {
			continue
		}
		var stamp string
		if json.Unmarshal(value, &stamp) != nil {
			continue
		}
		if cutoff, err := time.Parse(time.RFC3339Nano, stamp); err == nil {
			cutoffs[key[len(prefix):]] = cutoff
		}
	}
	return cutoffs
}

func CountUnreviewed(rows []store.Trade, journals map[string]store.TradeJournal, olderThan time.Time, cutoffs map[string]time.Time) int {
	count := 0
	for _, trade := range rows {
		if !trade.ClosedAt.Valid || !trade.ClosedAt.Time.Before(olderThan) {
			continue
		}
		if cutoff, ok := cutoffs[trade.AccountID]; ok && !trade.ClosedAt.Time.After(cutoff) {
			continue
		}
		if !Graded(journals[trade.ID].TradeQuality) {
			count++
		}
	}
	return count
}
