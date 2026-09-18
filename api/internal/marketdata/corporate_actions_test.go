package marketdata

import (
	"testing"
	"time"
)

func TestFindCorporateActionCandidates(t *testing.T) {
	tests := []struct {
		name      string
		status    string
		bars      []Bar
		wantType  string
		wantRatio float64
		wantDate  string
		wantCount int
	}{
		{
			name: "normal series", status: "unadjusted", wantCount: 0,
			bars: []Bar{dailyBar("2026-01-01", 100, 104, 98, 102), dailyBar("2026-01-02", 103, 106, 101, 105)},
		},
		{
			name: "two for one split", status: "unadjusted", wantCount: 1,
			bars:     []Bar{dailyBar("2026-01-01", 100, 104, 98, 102), dailyBar("2026-01-02", 51, 52, 49, 50)},
			wantType: "stock_split", wantRatio: 2,
		},
		{
			name: "reverse split", status: "unadjusted", wantCount: 1,
			bars:     []Bar{dailyBar("2026-01-01", 10, 10.4, 9.8, 10.2), dailyBar("2026-01-02", 51, 52, 49, 50)},
			wantType: "reverse_stock_split", wantRatio: 5,
		},
		{
			name: "large genuine move stays uncertain", status: "unadjusted", wantCount: 1,
			bars:     []Bar{dailyBar("2026-01-01", 100, 108, 96, 102), dailyBar("2026-01-02", 64, 75, 61, 72)},
			wantType: "other_corporate_action", wantRatio: 1.594,
		},
		{
			name: "adjustment mismatch", status: "adjusted", wantCount: 1,
			bars:     []Bar{dailyBar("2026-01-01", 100, 104, 98, 102), dailyBar("2026-01-02", 51, 52, 49, 50)},
			wantType: "adjustment_mismatch", wantRatio: 2,
		},
		{
			name: "5803 adjusted prices with provider split event", status: "split_adjusted", wantCount: 1,
			bars: func() []Bar {
				before := dailyBar("2026-03-31", 4250, 4281, 4056, 4090)
				after := dailyBar("2026-04-01", 4370, 4470, 4291, 4446)
				after.SplitRatio = 6
				return []Bar{before, after}
			}(),
			wantType: "stock_split", wantRatio: 6, wantDate: "2026-04-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindCorporateActionCandidates(Response{
				Instrument: "7203", Source: "test", AdjustmentStatus: tt.status, Bars: tt.bars,
			})
			if len(got) != tt.wantCount {
				t.Fatalf("got %d candidates, want %d: %#v", len(got), tt.wantCount, got)
			}
			if tt.wantCount == 0 {
				return
			}
			if got[0].CandidateType != tt.wantType || got[0].SuspectedRatio != tt.wantRatio {
				t.Fatalf("got type=%q ratio=%v, want type=%q ratio=%v", got[0].CandidateType, got[0].SuspectedRatio, tt.wantType, tt.wantRatio)
			}
			if tt.wantDate != "" && got[0].EffectiveDate != tt.wantDate {
				t.Fatalf("got effective date %q, want %q", got[0].EffectiveDate, tt.wantDate)
			}
			if got[0].Status != "unconfirmed" || len(got[0].Evidence) < 1 {
				t.Fatalf("candidate must remain reviewable and unconfirmed: %#v", got[0])
			}
		})
	}
}

func TestCorporateActionWarnings(t *testing.T) {
	candidate := CorporateActionCandidate{
		Instrument: "7203", EffectiveDate: "2026-01-02", CandidateType: "stock_split",
		SuspectedRatio: 2, Status: "unconfirmed",
	}
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	if got := CorporateActionWarnings([]CorporateActionCandidate{candidate}, from, to); len(got) != 1 {
		t.Fatalf("affected range got warnings %v", got)
	}
	candidate.Status = "rejected"
	if got := CorporateActionWarnings([]CorporateActionCandidate{candidate}, from, to); len(got) != 0 {
		t.Fatalf("rejected candidate got warnings %v", got)
	}
}

func dailyBar(date string, open, high, low, close float64) Bar {
	t, _ := time.Parse("2006-01-02", date)
	return Bar{Time: t.Unix(), MarketDate: date, Open: open, High: high, Low: low, Close: close, Volume: 100}
}
