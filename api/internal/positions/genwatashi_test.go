package positions

import (
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func genwatashiLegs(qty, price, fees float64, reported map[string]any) []store.Execution {
	cash := execution("c", "a", "S", "sbi:cash", "sell", "2026-09-03T00:00:00Z", qty, price)
	short := execution("s", "a", "S", "sbi:margin-short", "buy", "2026-09-03T00:00:00.000001Z", qty, price)
	cash.Fees = fees
	for _, ex := range []*store.Execution{&cash, &short} {
		d := map[string]any{"lot": lotFromDetails(*ex), "settlement_type": "genwatashi", "settlement_id": "event"}
		if ex.ID == "c" {
			for k, v := range reported {
				d[k] = v
			}
		}
		b, _ := json.Marshal(d)
		ex.Details = sql.NullString{String: string(b), Valid: true}
	}
	return []store.Execution{cash, short}
}

func TestGenwatashiReplayCostsOnceAndIsolation(t *testing.T) {
	cash := execution("1", "a", "S", "sbi:cash", "buy", "2026-09-01T00:00:00Z", 200, 900)
	short := execution("2", "a", "S", "sbi:margin-short", "sell", "2026-09-02T00:00:00Z", 200, 1000)
	short.Fees = 200
	other := execution("other", "b", "S", "sbi:cash", "buy", "2026-09-01T00:00:00Z", 100, 500)
	long := execution("long", "a", "S", "sbi:margin-long", "buy", "2026-09-01T00:00:00Z", 50, 700)
	fills := append([]store.Execution{cash, short, other, long}, genwatashiLegs(100, 1000, 600, map[string]any{"broker_reported_settlement_proceeds": 99400.0})...)
	snap := Replay(fills, []time.Time{day("2026-09-03")})[0]
	require.Empty(t, snap.Warnings)
	require.Len(t, snap.Accounts, 2)
	require.Equal(t, 9400.0, snap.Accounts[0].RealizedPnL)
	require.Equal(t, -80700.0, snap.Accounts[0].CashDelta) // -180000 -200 +99400 +100
	require.Equal(t, 700.0, snap.Accounts[0].Fees)         // 100 unclosed +600 settled
	require.Len(t, snap.Accounts[0].Positions, 3)
	require.Equal(t, 100.0, snap.Accounts[1].Positions[0].Quantity)
	results := SettlementResults(fills)
	require.Len(t, results, 2)
	require.Equal(t, 600.0, results["c"].ApplicableCosts)
}

func TestGenwatashiReplayRejectsBrokenPairAtomically(t *testing.T) {
	for _, mutate := range []func([]store.Execution) []store.Execution{
		func(x []store.Execution) []store.Execution { return x[:1] },
		func(x []store.Execution) []store.Execution { x[1].Symbol = "OTHER"; return x },
		func(x []store.Execution) []store.Execution { x[1].Quantity = 50; return x },
		func(x []store.Execution) []store.Execution { x[1].AccountID = "b"; return x },
	} {
		fills := []store.Execution{execution("1", "a", "S", "sbi:cash", "buy", "2026-09-01T00:00:00Z", 100, 900), execution("2", "a", "S", "sbi:margin-short", "sell", "2026-09-02T00:00:00Z", 100, 1000)}
		fills = append(fills, mutate(genwatashiLegs(100, 1000, 0, nil))...)
		snap := Replay(fills, []time.Time{day("2026-09-03")})[0]
		require.NotEmpty(t, snap.Warnings)
		require.Len(t, snap.Accounts[0].Positions, 2)
		require.Zero(t, snap.Accounts[0].RealizedPnL)
		require.Empty(t, SettlementResults(fills))
	}
}

func TestGenwatashiReportedShortEntryReducesRemainingBasis(t *testing.T) {
	fills := []store.Execution{execution("1", "a", "S", "sbi:cash", "buy", "2026-09-01T00:00:00Z", 200, 900), execution("2", "a", "S", "sbi:margin-short", "sell", "2026-09-01T00:00:00Z", 100, 1000), execution("3", "a", "S", "sbi:margin-short", "sell", "2026-09-02T00:00:00Z", 100, 1200)}
	fills = append(fills, genwatashiLegs(100, 1000, 0, map[string]any{"broker_reported_settlement_proceeds": 100000.0})...)
	snap := Replay(fills, []time.Time{day("2026-09-03")})[0]
	require.Empty(t, snap.Warnings)
	require.Equal(t, 1200.0, snap.Accounts[0].Positions[1].AverageCost)
	require.Equal(t, 10000.0, snap.Accounts[0].RealizedPnL)
}
