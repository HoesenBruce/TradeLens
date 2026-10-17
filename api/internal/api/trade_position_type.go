package api

import (
	"encoding/json"

	"github.com/tradermemos/api/internal/store"
)

// Only explicit SBI buckets establish a type; incomplete or mixed evidence stays unknown.
func tradePositionType(fills []store.ListOptionExecutionDetailsForUserRow) string {
	lot := ""
	for _, f := range fills {
		var d struct {
			Lot          string `json:"lot"`
			PositionType string `json:"position_type"`
		}
		if !f.Details.Valid || json.Unmarshal([]byte(f.Details.String), &d) != nil {
			return "unknown"
		}
		expected := ""
		switch d.Lot {
		case "sbi:cash":
			expected = "cash"
		case "sbi:margin-long":
			expected = "margin_long"
		case "sbi:margin-short":
			expected = "margin_short"
		default:
			return "unknown"
		}
		if (d.PositionType != "" && d.PositionType != expected) || (lot != "" && lot != d.Lot) {
			return "unknown"
		}
		lot = d.Lot
	}
	switch lot {
	case "sbi:cash":
		return "cash"
	case "sbi:margin-long", "sbi:margin-short":
		return "margin"
	default:
		return "unknown"
	}
}
