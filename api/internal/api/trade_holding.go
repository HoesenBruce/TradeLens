package api

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

// Holding is a read model: never replace date precision with elapsed seconds
// or change the persisted aggregation boundaries.
type tradeHoldingDTO struct {
	Kind         string `json:"kind"`
	Seconds      *int64 `json:"seconds"`
	CalendarDays *int   `json:"calendar_days"`
	TimeZone     string `json:"time_zone"`
	Ongoing      bool   `json:"ongoing"`
}

func tradeHolding(t store.Trade, fills []store.ListOptionExecutionDetailsForUserRow, now time.Time) tradeHoldingDTO {
	h := tradeHoldingDTO{Kind: "unknown", Ongoing: t.Status == "open",
		TimeZone: marketdata.MarketTimezone(marketdata.Request{Symbol: t.Symbol, InstrumentType: t.InstrumentType})}
	if t.OpenedAt.IsZero() || t.QtyOpened <= 0 || strings.Contains(t.AccountingWarning, "Incomplete SBI opening evidence") {
		return h
	}
	end := now
	if !h.Ongoing {
		if t.Status != "closed" || !t.ClosedAt.Valid {
			return h
		}
		end = t.ClosedAt.Time
	}
	startFound, endFound, dateOnly := false, h.Ongoing, false
	for _, f := range fills {
		start, finish := f.ExecutedAt.Equal(t.OpenedAt), !h.Ongoing && f.ExecutedAt.Equal(end)
		if !start && !finish {
			continue
		}
		var details struct {
			Precision string `json:"source_time_precision"`
		}
		if f.Details.Valid && f.Details.String != "" && json.Unmarshal([]byte(f.Details.String), &details) != nil {
			return h
		}
		if details.Precision != "" && details.Precision != "timestamp" && details.Precision != "date" {
			return h
		}
		startFound, endFound = startFound || start, endFound || finish
		dateOnly = dateOnly || details.Precision == "date"
	}
	if !startFound || !endFound {
		return h
	}
	if dateOnly {
		loc, err := time.LoadLocation(h.TimeZone)
		if err != nil {
			return h
		}
		// Compare civil dates in UTC so DST days still count as one day.
		civil := func(at time.Time) time.Time {
			y, m, d := at.In(loc).Date()
			return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
		}
		days := int(civil(end).Sub(civil(t.OpenedAt)).Hours() / 24)
		if days < 0 {
			return h
		}
		h.Kind, h.CalendarDays = "calendar", &days
	} else if !end.Before(t.OpenedAt) {
		secs := int64(end.Sub(t.OpenedAt).Seconds())
		h.Kind, h.Seconds = "elapsed", &secs
	}
	return h
}
