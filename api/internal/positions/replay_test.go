package positions

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func execution(id, account, symbol, lot, side, at string, quantity, price float64) store.Execution {
	timestamp, _ := time.Parse(time.RFC3339, at)
	details := sql.NullString{}
	if lot != "" {
		details = sql.NullString{String: `{"lot":"` + lot + `"}`, Valid: true}
	}
	return store.Execution{
		ID: id, AccountID: account, Symbol: symbol, InstrumentType: "stock", Side: side,
		Quantity: quantity, Price: price, ExecutedAt: timestamp, Multiplier: 1, Details: details,
	}
}

func day(value string) time.Time {
	date, _ := time.Parse(time.DateOnly, value)
	return date
}

func TestReplayCashLongPreservesOpeningStateAndPartialClose(t *testing.T) {
	executions := []store.Execution{
		execution("1", "a", "5401", "sbi:cash", "buy", "2026-08-10T01:00:00Z", 100, 10),
		execution("2", "a", "5401", "sbi:cash", "buy", "2026-08-11T01:00:00Z", 100, 20),
		execution("3", "a", "5401", "sbi:cash", "sell", "2026-09-02T01:00:00Z", 40, 25),
		execution("4", "a", "5401", "sbi:cash", "sell", "2026-09-03T01:00:00Z", 160, 30),
	}

	snapshots := Replay(executions, []time.Time{day("2026-09-03"), day("2026-09-01"), day("2026-09-02")})
	require.Equal(t, []string{"2026-09-01", "2026-09-02", "2026-09-03"}, []string{snapshots[0].Date, snapshots[1].Date, snapshots[2].Date})
	require.Equal(t, 200.0, snapshots[0].Accounts[0].Positions[0].Quantity)
	require.Equal(t, 15.0, snapshots[0].Accounts[0].Positions[0].AverageCost)
	require.Equal(t, 160.0, snapshots[1].Accounts[0].Positions[0].Quantity)
	require.Equal(t, 400.0, snapshots[1].Accounts[0].RealizedPnL)
	require.Empty(t, snapshots[2].Accounts[0].Positions)
	require.Equal(t, 2800.0, snapshots[2].Accounts[0].RealizedPnL)
	require.Equal(t, 2800.0, snapshots[2].Accounts[0].CashDelta)
}

func TestReplayMarginLongAndShort(t *testing.T) {
	executions := []store.Execution{
		execution("1", "a", "L", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 10, 100),
		execution("2", "a", "S", "sbi:margin-short", "sell", "2026-09-01T01:00:00Z", 8, 200),
		execution("3", "a", "L", "sbi:margin-long", "sell", "2026-09-02T01:00:00Z", 4, 120),
		execution("4", "a", "S", "sbi:margin-short", "buy", "2026-09-02T01:00:00Z", 3, 180),
		execution("5", "a", "L", "sbi:margin-long", "sell", "2026-09-03T01:00:00Z", 6, 90),
		execution("6", "a", "S", "sbi:margin-short", "buy", "2026-09-03T01:00:00Z", 5, 210),
	}

	snapshots := Replay(executions, []time.Time{day("2026-09-02"), day("2026-09-03")})
	require.Equal(t, []Position{
		{AccountID: "a", Symbol: "L", InstrumentType: "stock", Kind: MarginLong, Lot: "sbi:margin-long", Quantity: 6, AverageCost: 100, Multiplier: 1},
		{AccountID: "a", Symbol: "S", InstrumentType: "stock", Kind: MarginShort, Lot: "sbi:margin-short", Quantity: 5, AverageCost: 200, Multiplier: 1},
	}, snapshots[0].Accounts[0].Positions)
	require.Equal(t, 140.0, snapshots[0].Accounts[0].RealizedPnL)
	require.Empty(t, snapshots[1].Accounts[0].Positions)
	require.Equal(t, 30.0, snapshots[1].Accounts[0].RealizedPnL)
	require.Equal(t, 30.0, snapshots[1].Accounts[0].CashDelta)
}

func withReportedPnl(ex store.Execution, pnl float64) store.Execution {
	details := map[string]any{"lot": lotFromDetails(ex), "broker_reported_realized_pnl": pnl, "realized_pnl_source": "broker_reported"}
	encoded, _ := json.Marshal(details)
	ex.Details = sql.NullString{String: string(encoded), Valid: true}
	return ex
}

func TestReplaySBIMarginReportedCloseAndMissingFallback(t *testing.T) {
	first := execution("open-a", "a", "AAA", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 100, 1000)
	first.Fees = 5
	second := execution("open-b", "a", "AAA", "sbi:margin-long", "buy", "2026-09-02T01:00:00Z", 100, 1100)
	second.Fees = 5
	close := withReportedPnl(execution("close", "a", "AAA", "sbi:margin-long", "sell", "2026-09-03T01:00:00Z", 100, 1090), 8993)
	close.Fees = 2
	cash := execution("cash", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 10, 500)
	snapshot := Replay([]store.Execution{close, first, second, cash}, []time.Time{day("2026-09-03")})[0]
	require.Empty(t, snapshot.Warnings)
	require.Equal(t, 8993.0, snapshot.Accounts[0].RealizedPnL)
	require.Equal(t, 12.0, snapshot.Accounts[0].Fees)
	require.Equal(t, []Kind{CashLong, MarginLong}, []Kind{snapshot.Accounts[0].Positions[0].Kind, snapshot.Accounts[0].Positions[1].Kind})
	require.Equal(t, 100.0, snapshot.Accounts[0].Positions[1].Quantity)

	missing := close
	missing.Details = sql.NullString{String: `{"lot":"sbi:margin-long"}`, Valid: true}
	snapshot = Replay([]store.Execution{first, second, missing}, []time.Time{day("2026-09-03")})[0]
	require.Equal(t, "margin_realized_pnl_unavailable", snapshot.Warnings[0].Code)
	require.Zero(t, snapshot.Accounts[0].RealizedPnL)
}

func TestReplaySBIMarginShortReportedSignAndSingleOpenFallback(t *testing.T) {
	open := execution("open", "a", "AAA", "sbi:margin-short", "sell", "2026-09-01T01:00:00Z", 20, 300)
	open.Fees = 6
	close := withReportedPnl(execution("close", "a", "AAA", "sbi:margin-short", "buy", "2026-09-02T01:00:00Z", 20, 280), 388)
	close.Fees = 6
	snapshot := Replay([]store.Execution{open, close}, []time.Time{day("2026-09-02")})[0]
	require.Equal(t, 388.0, snapshot.Accounts[0].RealizedPnL)
	require.Equal(t, 388.0, snapshot.Accounts[0].CashDelta)

	close.Details = sql.NullString{String: `{"lot":"sbi:margin-short"}`, Valid: true}
	snapshot = Replay([]store.Execution{open, close}, []time.Time{day("2026-09-02")})[0]
	require.Empty(t, snapshot.Warnings)
	require.Equal(t, 388.0, snapshot.Accounts[0].RealizedPnL)
}

func TestReplaySBIMarginPartialAndCompleteReportedSettlements(t *testing.T) {
	open := execution("open", "a", "AAA", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 10, 100)
	open.Fees = 5
	first := withReportedPnl(execution("first", "a", "AAA", "sbi:margin-long", "sell", "2026-09-02T01:00:00Z", 5, 110), 46.5)
	first.Fees = 1
	last := withReportedPnl(execution("last", "a", "AAA", "sbi:margin-long", "sell", "2026-09-03T01:00:00Z", 5, 90), -53.5)
	last.Fees = 1
	snapshots := Replay([]store.Execution{last, first, open}, []time.Time{day("2026-09-02"), day("2026-09-03")})
	require.Equal(t, 46.5, snapshots[0].Accounts[0].RealizedPnL)
	require.Equal(t, 5.0, snapshots[0].Accounts[0].Positions[0].Quantity)
	require.Equal(t, -7.0, snapshots[1].Accounts[0].RealizedPnL)
	require.Equal(t, -7.0, snapshots[1].Accounts[0].CashDelta)
	require.Empty(t, snapshots[1].Accounts[0].Positions)
}

func TestReplayGenbikiMovesMarginLongToCashOnce(t *testing.T) {
	open := execution("1", "a", "5401", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 100, 1000)
	closeMargin := execution("2", "a", "5401", "sbi:margin-long", "sell", "2026-09-02T01:00:00Z", 100, 1090)
	closeMargin.Fees = 500
	closeMargin.Details = sql.NullString{String: `{"lot":"sbi:margin-long","event_type":"position_conversion","conversion_type":"genbiki","conversion_id":"c1"}`, Valid: true}
	openCash := execution("3", "a", "5401", "sbi:cash", "buy", "2026-09-02T01:00:00.000001Z", 100, 1090)
	openCash.Details = sql.NullString{String: `{"lot":"sbi:cash","event_type":"position_conversion","conversion_type":"genbiki","conversion_id":"c1"}`, Valid: true}

	snapshot := Replay([]store.Execution{openCash, closeMargin, open}, []time.Time{day("2026-09-02")})[0]
	require.Equal(t, []Position{{
		AccountID: "a", Symbol: "5401", InstrumentType: "stock", Kind: CashLong,
		Lot: "sbi:cash", Quantity: 100, AverageCost: 1005, Multiplier: 1,
	}}, snapshot.Accounts[0].Positions)
	require.Zero(t, snapshot.Accounts[0].RealizedPnL)
	require.Equal(t, -500.0, snapshot.Accounts[0].CashDelta)
	require.Equal(t, 500.0, snapshot.Accounts[0].Fees)
}

func TestReplayGenbikiMergesIntoExistingCashAtAverageCost(t *testing.T) {
	cash := execution("0", "a", "5401", "sbi:cash", "buy", "2026-08-31T01:00:00Z", 100, 900)
	open := execution("1", "a", "5401", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 100, 1000)
	close := execution("2", "a", "5401", "sbi:margin-long", "sell", "2026-09-02T01:00:00Z", 100, 1090)
	close.Fees = 500
	close.Details = sql.NullString{String: `{"lot":"sbi:margin-long","conversion_type":"genbiki","conversion_id":"c1"}`, Valid: true}
	converted := execution("3", "a", "5401", "sbi:cash", "buy", "2026-09-02T01:00:00.000001Z", 100, 1090)
	converted.Details = sql.NullString{String: `{"lot":"sbi:cash","conversion_type":"genbiki","conversion_id":"c1"}`, Valid: true}
	snapshot := Replay([]store.Execution{converted, close, open, cash}, []time.Time{day("2026-09-02")})[0]
	require.Equal(t, 200.0, snapshot.Accounts[0].Positions[0].Quantity)
	require.Equal(t, 953.0, snapshot.Accounts[0].Positions[0].AverageCost)
	require.Zero(t, snapshot.Accounts[0].RealizedPnL)
}

func TestReplayIsolatesAccountsInstrumentsAndRejectsOverClose(t *testing.T) {
	executions := []store.Execution{
		execution("1", "b", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 1, 10),
		execution("2", "a", "BBB", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 2, 20),
		execution("3", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 3, 30),
		execution("4", "a", "AAA", "sbi:cash", "sell", "2026-09-02T01:00:00Z", 4, 40),
	}

	snapshot := Replay(executions, []time.Time{day("2026-09-02"), day("2026-09-02")})[0]
	require.Equal(t, []string{"a", "b"}, []string{snapshot.Accounts[0].AccountID, snapshot.Accounts[1].AccountID})
	require.Equal(t, []string{"AAA", "BBB"}, []string{
		snapshot.Accounts[0].Positions[0].Symbol, snapshot.Accounts[0].Positions[1].Symbol,
	})
	require.Equal(t, "AAA", snapshot.Accounts[1].Positions[0].Symbol)
	require.Equal(t, 3.0, snapshot.Accounts[0].Positions[0].Quantity)
	require.Equal(t, -130.0, snapshot.Accounts[0].CashDelta)
	require.Equal(t, -10.0, snapshot.Accounts[1].CashDelta)
	require.Equal(t, "invalid_execution_sequence", snapshot.Warnings[0].Code)
	require.Equal(t, "AAA", snapshot.Warnings[0].Instrument)
	require.Equal(t, "4", snapshot.Warnings[0].ExecutionID)
	require.Equal(t, "2026-09-02", snapshot.Warnings[0].Date)
}

func TestReplayOrdersSBIOpensBeforeSameDayCloses(t *testing.T) {
	date := "2026-09-02"
	snapshot := Replay([]store.Execution{
		execution("sell", "a", "7203", "sbi:cash", "sell", date+"T01:00:00Z", 100, 101),
		execution("buy", "a", "7203", "sbi:cash", "buy", date+"T01:00:00.000001Z", 100, 100),
	}, []time.Time{day(date)})[0]

	require.Empty(t, snapshot.Warnings)
	require.Equal(t, 100.0, snapshot.Accounts[0].RealizedPnL)
}

func TestReplaySBICashDailyCostBasis(t *testing.T) {
	tests := []struct {
		name              string
		fills             []store.Execution
		pnl, qty, average float64
	}{
		{"multi-day partial", []store.Execution{
			execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 100, 1000),
			execution("2", "a", "AAA", "sbi:cash", "buy", "2026-09-02T01:00:00Z", 100, 1100),
			execution("3", "a", "AAA", "sbi:cash", "sell", "2026-09-03T01:00:00Z", 100, 1090),
		}, 4000, 100, 1050},
		{"buy sell buy", []store.Execution{
			execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-03T00:10:00Z", 100, 1000),
			execution("2", "a", "AAA", "sbi:cash", "sell", "2026-09-03T01:30:00Z", 100, 1100),
			execution("3", "a", "AAA", "sbi:cash", "buy", "2026-09-03T05:00:00Z", 100, 1200),
		}, 0, 100, 1100},
		{"previous holding sell buy", []store.Execution{
			execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-02T01:00:00Z", 100, 900),
			execution("2", "a", "AAA", "sbi:cash", "sell", "2026-09-03T00:10:00Z", 100, 1100),
			execution("3", "a", "AAA", "sbi:cash", "buy", "2026-09-03T05:00:00Z", 100, 1200),
		}, 5000, 100, 1050},
		{"multiple buys sells", []store.Execution{
			execution("1", "a", "AAA", "sbi:cash", "buy", "2026-09-03T00:10:00Z", 100, 1000),
			execution("2", "a", "AAA", "sbi:cash", "sell", "2026-09-03T01:00:00Z", 50, 1100),
			execution("3", "a", "AAA", "sbi:cash", "buy", "2026-09-03T02:00:00Z", 100, 1200),
			execution("4", "a", "AAA", "sbi:cash", "sell", "2026-09-03T05:00:00Z", 50, 1300),
		}, 10000, 100, 1100},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			snapshot := Replay(tc.fills, []time.Time{day("2026-09-03")})[0]
			require.Empty(t, snapshot.Warnings)
			require.Equal(t, tc.pnl, snapshot.Accounts[0].RealizedPnL)
			require.Equal(t, tc.qty, snapshot.Accounts[0].Positions[0].Quantity)
			require.Equal(t, tc.average, snapshot.Accounts[0].Positions[0].AverageCost)
		})
	}
}

func TestReplaySBICashAcquisitionFeesAndRounding(t *testing.T) {
	buy := execution("buy", "a", "AAA", "sbi:cash", "buy", "2026-09-01T01:00:00Z", 100, 120)
	buy.Fees = 55
	sell := execution("sell", "a", "AAA", "sbi:cash", "sell", "2026-09-02T01:00:00Z", 40, 123)
	sell.Fees = 10
	snapshot := Replay([]store.Execution{sell, buy}, []time.Time{day("2026-09-02")})[0]
	require.Equal(t, 121.0, snapshot.Accounts[0].Positions[0].AverageCost)
	require.Equal(t, 70.0, snapshot.Accounts[0].RealizedPnL)
	require.Equal(t, 65.0, snapshot.Accounts[0].Fees)
	require.Equal(t, -7145.0, snapshot.Accounts[0].CashDelta)
}

func TestReplayAppliesExplicitSplitBeforeEffectiveDateExecutions(t *testing.T) {
	snapshot := ReplayWithSplits([]store.Execution{
		execution("open", "a", "7013", "sbi:margin-long", "buy", "2025-09-25T01:00:00Z", 100, 17500),
		execution("close", "a", "7013", "sbi:margin-long", "sell", "2025-09-29T01:00:00Z", 700, 2500),
	}, []time.Time{day("2025-09-29")}, []Split{{
		Symbol: "7013", InstrumentType: "stock", EffectiveDate: day("2025-09-29"), Ratio: 7,
	}})[0]

	require.Empty(t, snapshot.Warnings)
	require.Empty(t, snapshot.Accounts[0].Positions)
	require.Zero(t, snapshot.Accounts[0].RealizedPnL)
}
