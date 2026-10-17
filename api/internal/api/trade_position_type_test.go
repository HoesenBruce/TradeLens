package api

import (
	"database/sql"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func TestTradePositionType(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		details    []string
	}{
		{"cash", "cash", []string{`{"lot":"sbi:cash","position_type":"cash"}`, `{"lot":"sbi:cash"}`}},
		{"margin long", "margin", []string{`{"lot":"sbi:margin-long","position_type":"margin_long"}`}},
		{"margin short", "margin", []string{`{"lot":"sbi:margin-short"}`}},
		{"conversion cash", "cash", []string{`{"lot":"sbi:cash","conversion_type":"genbiki"}`}},
		{"conversion margin", "margin", []string{`{"lot":"sbi:margin-long","conversion_type":"genbiki"}`}},
		{"delivery", "margin", []string{`{"lot":"sbi:margin-short","settlement_type":"genwatashi"}`}},
		{"no fills", "unknown", nil},
		{"missing", "unknown", []string{""}},
		{"null", "unknown", []string{"null"}},
		{"malformed", "unknown", []string{"{"}},
		{"metadata alone", "unknown", []string{`{"position_type":"cash"}`}},
		{"non SBI", "unknown", []string{`{"lot":"ibkr:cash"}`}},
		{"unrecognized SBI", "unknown", []string{`{"lot":"sbi:other"}`}},
		{"conflicting fields", "unknown", []string{`{"lot":"sbi:cash","position_type":"margin_long"}`}},
		{"mixed types", "unknown", []string{`{"lot":"sbi:cash"}`, `{"lot":"sbi:margin-long"}`}},
		{"mixed directions", "unknown", []string{`{"lot":"sbi:margin-short"}`, `{"lot":"sbi:margin-long"}`}},
		{"incomplete evidence", "unknown", []string{`{"lot":"sbi:cash"}`, `{}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fills := []store.ListOptionExecutionDetailsForUserRow{}
			for _, d := range tc.details {
				fills = append(fills, store.ListOptionExecutionDetailsForUserRow{Details: sql.NullString{String: d, Valid: d != ""}})
			}
			require.Equal(t, tc.want, tradePositionType(fills))
		})
	}
}
