package marketdata

import (
	"embed"
	"encoding/csv"
	"errors"
	"strings"
	"sync"
	"time"
)

//go:embed calendars/*.csv
var calendarFiles embed.FS

const CalendarVersion = "exchange-calendars-4.13.2-2020-2030"

type Session struct {
	Date  string    `json:"date"`
	Open  time.Time `json:"open"`
	Close time.Time `json:"close"`
}
type Calendar struct {
	Market   string    `json:"market"`
	Timezone string    `json:"timezone"`
	Currency string    `json:"currency"`
	Version  string    `json:"version"`
	Sessions []Session `json:"sessions"`
}

var calendarOnce sync.Once
var calendars map[string]Calendar
var calendarError error

// TradingCalendar returns versioned exchange sessions, never inferred from bars or weekdays.
func TradingCalendar(market string) (Calendar, error) {
	calendarOnce.Do(func() {
		calendars = map[string]Calendar{}
		for _, m := range []string{"JP", "US"} {
			c := Calendar{Market: m, Version: CalendarVersion, Timezone: "Asia/Tokyo", Currency: "JPY"}
			if m == "US" {
				c.Timezone = "America/New_York"
				c.Currency = "USD"
			}
			raw, err := calendarFiles.ReadFile("calendars/" + m + ".csv")
			if err != nil {
				calendarError = err
				return
			}
			rows, err := csv.NewReader(strings.NewReader(string(raw))).ReadAll()
			if err != nil {
				calendarError = err
				return
			}
			for _, r := range rows[1:] {
				if len(r) != 3 {
					calendarError = errors.New("invalid calendar row")
					return
				}
				open, err := time.Parse(time.RFC3339, r[1])
				if err != nil {
					calendarError = err
					return
				}
				close, err := time.Parse(time.RFC3339, r[2])
				if err != nil {
					calendarError = err
					return
				}
				c.Sessions = append(c.Sessions, Session{Date: r[0], Open: open.UTC(), Close: close.UTC()})
			}
			calendars[m] = c
		}
	})
	if calendarError != nil {
		return Calendar{}, calendarError
	}
	c, ok := calendars[market]
	if !ok {
		return Calendar{}, errors.New("calendar unavailable")
	}
	return c, nil
}
