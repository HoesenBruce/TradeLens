package accountvalue

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
	"github.com/tradermemos/api/internal/store"
)

func TestReconstructValuesAccountsPositionsAndCashEvents(t *testing.T) {
	service := testService(map[string]marketdata.Response{
		"AAA": bars("AAA", "unadjusted", map[string]float64{"2026-09-01": 10, "2026-09-02": 12, "2026-09-03": 15}),
		"L":   bars("L", "unadjusted", map[string]float64{"2026-09-01": 100, "2026-09-02": 110, "2026-09-03": 90}),
		"S":   bars("S", "unadjusted", map[string]float64{"2026-09-01": 200, "2026-09-02": 180, "2026-09-03": 210}),
	})
	executions := []store.Execution{
		execution("1", "a", "AAA", "sbi:cash", "buy", "2026-08-30T01:00:00Z", 10, 10),
		execution("2", "a", "AAA", "sbi:cash", "sell", "2026-09-02T01:00:00Z", 4, 12),
		execution("3", "a", "AAA", "sbi:cash", "sell", "2026-09-03T01:00:00Z", 6, 15),
		execution("4", "a", "L", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 2, 100),
		execution("5", "a", "S", "sbi:margin-short", "sell", "2026-09-01T01:00:00Z", 3, 200),
		execution("6", "b", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
	}
	cash := []store.CashTransaction{
		cashRow("1", "a", "deposit", 1000, "2026-08-01T01:00:00Z"),
		cashRow("2", "a", "withdrawal", -100, "2026-09-02T02:00:00Z"),
		cashRow("3", "a", "dividend", 10, "2026-09-02T03:00:00Z"),
		cashRow("4", "a", "fee", -5, "2026-09-02T04:00:00Z"),
		cashRow("5", "a", "adjustment", -2, "2026-09-02T05:00:00Z"),
		cashRow("7", "a", "interest", 3, "2026-09-02T06:00:00Z"),
		cashRow("6", "b", "deposit", 100, "2026-08-01T01:00:00Z"),
	}

	result, err := service.Reconstruct(context.Background(), Request{
		Executions: executions, CashTransactions: cash,
		MarketSessions: []time.Time{day("2026-09-01"), day("2026-09-02"), day("2026-09-03")},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"a", "b"}, []string{result.Accounts[0].AccountID, result.Accounts[1].AccountID})
	a := result.Accounts[0].Points
	require.Equal(t, 1000.0, *a[0].EstimatedAccountValue)
	require.Equal(t, 900.0, a[1].ContributedCapital)
	require.Equal(t, 3.0, a[1].RealizedPnL)
	require.Equal(t, 152.0, *a[1].OpenPositionValue)
	require.Equal(t, 92.0, *a[1].UnrealizedPnL)
	require.Equal(t, 1006.0, *a[1].EstimatedAccountValue)
	require.Equal(t, 944.0, a[2].CashBalance)
	require.Equal(t, -50.0, *a[2].OpenPositionValue)
	require.Equal(t, 894.0, *a[2].EstimatedAccountValue)
	require.Equal(t, 100.0, *result.Accounts[1].Points[0].EstimatedAccountValue)
}

func TestReconstructIdentifiesInvalidExecution(t *testing.T) {
	service := testService(map[string]marketdata.Response{
		"AAA": bars("AAA", "unadjusted", map[string]float64{"2026-09-02": 10}),
	})
	result, err := service.Reconstruct(context.Background(), Request{
		Executions: []store.Execution{
			execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
			execution("2", "a", "AAA", "sbi:cash", "sell", "2026-09-02T01:00:00Z", 2, 10),
		},
		MarketSessions: []time.Time{day("2026-09-02")},
	})
	require.NoError(t, err)
	warning := result.Accounts[0].Points[0].Warnings[0]
	require.Equal(t, "AAA", warning.Instrument)
	require.Equal(t, "2", warning.ExecutionID)
	require.Equal(t, "2026-09-02", warning.Date)
}

func TestReconstructMissingSuspendedAndCorporateActionPrices(t *testing.T) {
	service := testService(map[string]marketdata.Response{
		"AAA": bars("AAA", "unadjusted", map[string]float64{"2026-09-01": 10}),
		"BBB": bars("BBB", "unadjusted", map[string]float64{"2026-09-01": 100, "2026-09-02": 50}),
	})
	executions := []store.Execution{
		execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
		execution("2", "a", "BBB", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 100),
	}

	result, err := service.Reconstruct(context.Background(), Request{
		Executions: executions, MarketSessions: []time.Time{day("2026-09-02")},
		ConfirmedSuspensions: map[Instrument]map[string]bool{{"AAA", "stock"}: {"2026-09-02": true}},
	})
	require.NoError(t, err)
	point := result.Accounts[0].Points[0]
	require.Equal(t, "unsupported_corporate_action", point.Status)
	require.Nil(t, point.EstimatedAccountValue)
	require.Contains(t, warningCodes(point.Warnings), "carried_forward_suspension_price")
	require.Contains(t, warningCodes(point.Warnings), "unsupported_corporate_action")
}

func TestReconstructAppliesReportedSplit(t *testing.T) {
	response := bars("7013", "unadjusted", map[string]float64{"2025-09-26": 17500, "2025-09-29": 2500})
	response.Bars[1].SplitRatio = 7
	service := testService(map[string]marketdata.Response{"7013": response})
	result, err := service.Reconstruct(context.Background(), Request{
		Executions: []store.Execution{
			execution("open", "a", "7013", "sbi:margin-long", "buy", "2025-09-25T01:00:00Z", 100, 17500),
			execution("close", "a", "7013", "sbi:margin-long", "sell", "2025-09-29T01:00:00Z", 700, 2500),
		},
		MarketSessions: []time.Time{day("2025-09-29")},
	})

	require.NoError(t, err)
	point := result.Accounts[0].Points[0]
	require.Equal(t, "complete", point.Status)
	require.Empty(t, point.Warnings)
}

func TestSplitWithPartialClosePreservesAccountValue(t *testing.T) {
	response := bars("7013", "unadjusted", map[string]float64{
		"2025-09-26": 17745, "2025-09-29": 2535,
	})
	for i := range response.Bars {
		if response.Bars[i].MarketDate == "2025-09-29" {
			response.Bars[i].SplitRatio = 7
		}
	}
	result, err := testService(map[string]marketdata.Response{"7013": response}).Reconstruct(context.Background(), Request{
		Executions: []store.Execution{
			execution("buy", "a", "7013", "sbi:cash", "buy", "2025-09-25T01:00:00Z", 200, 17745),
			execution("sell", "a", "7013", "sbi:cash", "sell", "2025-09-29T01:00:00Z", 200, 2535),
		},
		CashTransactions: []store.CashTransaction{cashRow("deposit", "a", "deposit", 11000000, "2025-09-25T01:00:00Z")},
		MarketSessions:   []time.Time{day("2025-09-26"), day("2025-09-29")},
	})
	require.NoError(t, err)
	for _, point := range result.Accounts[0].Points {
		require.Equal(t, "complete", point.Status)
		require.InDelta(t, 11000000, *point.EstimatedAccountValue, 1)
		require.Equal(t, 11000000.0, point.ContributedCapital)
	}
}

func TestExplicitSplitsHonorsConfirmedAndRejected(t *testing.T) {
	responses := map[Instrument]marketdata.Response{
		{"AAA", "stock"}: {CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2025-09-29", CandidateType: "reverse_stock_split", SuspectedRatio: 5, Status: "confirmed"}}},
		{"BBB", "stock"}: {Bars: []marketdata.Bar{{MarketDate: "2025-09-29", SplitRatio: 7}}, CorporateActions: []marketdata.CorporateActionCandidate{{EffectiveDate: "2025-09-29", Status: "rejected"}}},
	}
	splits := explicitSplits(responses)
	require.Len(t, splits, 1)
	require.Equal(t, "AAA", splits[0].Symbol)
	require.Equal(t, 0.2, splits[0].Ratio)
}

func TestReconstructMissingPriceAndPartialProviderFailure(t *testing.T) {
	service := &Service{getBars: func(_ context.Context, req marketdata.Request) (marketdata.Response, error) {
		if req.Symbol == "BAD" {
			return marketdata.Response{}, errors.New("provider failed")
		}
		return bars(req.Symbol, "unadjusted", map[string]float64{}), nil
	}}
	executions := []store.Execution{
		execution("1", "a", "GOOD", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
		execution("2", "a", "BAD", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
	}

	result, err := service.Reconstruct(context.Background(), Request{Executions: executions, MarketSessions: []time.Time{day("2026-09-01")}})
	require.NoError(t, err)
	require.Equal(t, "incomplete_missing_price", result.Accounts[0].Points[0].Status)
	require.Len(t, result.Accounts[0].Points[0].Warnings, 2)

	service = &Service{getBars: func(context.Context, marketdata.Request) (marketdata.Response, error) {
		return marketdata.Response{}, errors.New("provider failed")
	}}
	_, err = service.Reconstruct(context.Background(), Request{Executions: executions[:1], MarketSessions: []time.Time{day("2026-09-01")}})
	require.ErrorContains(t, err, "provider failed")
}

func TestReconstructDoesNotInventWeekendPoints(t *testing.T) {
	service := testService(map[string]marketdata.Response{})
	result, err := service.Reconstruct(context.Background(), Request{
		CashTransactions: []store.CashTransaction{cashRow("1", "a", "deposit", 100, "2026-09-04T01:00:00Z")},
		MarketSessions:   []time.Time{day("2026-09-04"), day("2026-09-07")},
	})
	require.NoError(t, err)
	require.Equal(t, []string{"2026-09-04", "2026-09-07"}, []string{
		result.Accounts[0].Points[0].Date, result.Accounts[0].Points[1].Date,
	})
}

func testService(responses map[string]marketdata.Response) *Service {
	return &Service{getBars: func(_ context.Context, req marketdata.Request) (marketdata.Response, error) {
		return responses[req.Symbol], nil
	}}
}

func bars(symbol, adjustment string, closes map[string]float64) marketdata.Response {
	response := marketdata.Response{Instrument: symbol, Source: "test", AdjustmentStatus: adjustment}
	for date, close := range closes {
		at, _ := time.Parse(time.DateOnly, date)
		response.Bars = append(response.Bars, marketdata.Bar{
			Time: at.Unix(), MarketDate: date, Open: close, High: close, Low: close, Close: close,
		})
	}
	return response
}

func execution(id, account, symbol, lot, side, at string, quantity, price float64) store.Execution {
	timestamp, _ := time.Parse(time.RFC3339, at)
	return store.Execution{
		ID: id, AccountID: account, Symbol: symbol, InstrumentType: "stock", Side: side,
		Quantity: quantity, Price: price, ExecutedAt: timestamp, Multiplier: 1,
		Details: sql.NullString{String: `{"lot":"` + lot + `"}`, Valid: true},
	}
}

func cashRow(id, account, kind string, amount float64, at string) store.CashTransaction {
	timestamp, _ := time.Parse(time.RFC3339, at)
	return store.CashTransaction{ID: id, AccountID: account, Type: kind, Amount: amount, OccurredAt: timestamp}
}

func day(value string) time.Time {
	date, _ := time.Parse(time.DateOnly, value)
	return date
}

func warningCodes(warnings []Warning) []string {
	out := make([]string, len(warnings))
	for i, warning := range warnings {
		out[i] = warning.Code
	}
	return out
}
