package positions

import (
	"database/sql"
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

func TestReplayGenbikiMovesMarginLongToCashOnce(t *testing.T) {
	open := execution("1", "a", "5401", "sbi:margin-long", "buy", "2026-09-01T01:00:00Z", 100, 900)
	closeMargin := execution("2", "a", "5401", "sbi:margin-long", "sell", "2026-09-02T01:00:00Z", 100, 920)
	closeMargin.Fees = 20
	openCash := execution("3", "a", "5401", "sbi:cash", "buy", "2026-09-02T01:00:00.000001Z", 100, 920)

	snapshot := Replay([]store.Execution{openCash, closeMargin, open}, []time.Time{day("2026-09-02")})[0]
	require.Equal(t, []Position{{
		AccountID: "a", Symbol: "5401", InstrumentType: "stock", Kind: CashLong,
		Lot: "sbi:cash", Quantity: 100, AverageCost: 920, Multiplier: 1,
	}}, snapshot.Accounts[0].Positions)
	require.Equal(t, 1980.0, snapshot.Accounts[0].RealizedPnL)
	require.Equal(t, -90020.0, snapshot.Accounts[0].CashDelta)
	require.Equal(t, 20.0, snapshot.Accounts[0].Fees)
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
