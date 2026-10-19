package importer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/positions"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"
)

func syntheticSBIFill(id, side string, qty, price float64, hour int) store.Execution {
	at := time.Date(2032, 3, 2, hour, 0, 0, 0, sbiZone)
	effect := "increase"
	if side == "sell" {
		effect = "reduce"
	}
	source := map[string]string{"信用区分": "制度信用", "手数料/諸経費等": "0", "金利": "0", "貸株料": "0", "その他諸経費": "0", "original_executed_at": at.Format(time.RFC3339Nano)}
	data, _ := json.Marshal(map[string]any{"lot": "sbi:margin-long", "position_effect": effect, "sbi_source": source})
	return store.Execution{ID: id, UserID: "u", AccountID: "a", Symbol: "9101", Side: side, Quantity: qty, Price: price, InstrumentType: "stock", Multiplier: 1, ExecutedAt: at, Details: sql.NullString{String: string(data), Valid: true}}
}
func syntheticSBIReport(qty float64) SBIMarginPnLRow {
	pnl := qty * 10
	return SBIMarginPnLRow{Date: "2032/03/02", Symbol: "9101", Transaction: "信用返済売", Quantity: qty, Price: 110, Basis: 100, PnL: &pnl}
}
func changeSBIDetails(e *store.Execution, change func(map[string]any)) {
	d := sbiDetails(*e)
	change(d)
	v, _ := json.Marshal(d)
	e.Details.String = string(v)
}
func TestSBISettlementEvidenceBeforeAmbiguity(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(fmt.Sprint(short), func(t *testing.T) {
			open, a, b := syntheticSBIFill("open", "buy", 300, 100, 9), syntheticSBIFill("a", "sell", 100, 110, 11), syntheticSBIFill("b", "sell", 100, 110, 12)
			changeSBIDetails(&a, func(d map[string]any) { d["broker_reported_realized_pnl"] = 1000.0 })
			changeSBIDetails(&b, func(d map[string]any) { d["broker_reported_realized_pnl"] = 900.0 })
			row := syntheticSBIReport(100)
			if short {
				row.Transaction = "信用返済買"
				for _, e := range []*store.Execution{&open, &a, &b} {
					if e.Side == "sell" {
						e.Side = "buy"
					} else {
						e.Side = "sell"
					}
					changeSBIDetails(e, func(d map[string]any) { d["lot"] = "sbi:margin-short" })
				}
			}
			fills := []store.Execution{open, a, b}
			for range 2 {
				got := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, fills)[0]
				require.Equal(t, "enrichable", got.Status)
				require.Equal(t, "a", got.ExecutionID)
				require.Equal(t, "exact", got.MatchProvenance)
				require.Equal(t, "exact", got.OpeningMatch)
				slices.Reverse(fills)
			}
			changeSBIDetails(&b, func(d map[string]any) { delete(d, "broker_reported_realized_pnl") })
			require.Equal(t, "ambiguous_match", MatchSBIMarginPnL([]SBIMarginPnLRow{row}, []store.Execution{open, a, b})[0].Status)
			changeSBIDetails(&a, func(d map[string]any) { d["broker_reported_realized_pnl"] = -1.0 })
			changeSBIDetails(&b, func(d map[string]any) { d["broker_reported_realized_pnl"] = -2.0 })
			require.Equal(t, "conflict", MatchSBIMarginPnL([]SBIMarginPnLRow{row}, []store.Execution{open, a, b})[0].Status)
		})
	}
}

func TestSBIEquivalentCloseResultsRequireCompleteEvidence(t *testing.T) {
	for _, short := range []bool{false, true} {
		t.Run(fmt.Sprint(short), func(t *testing.T) {
			a, b := syntheticSBIFill("a", "sell", 100, 110, 11), syntheticSBIFill("b", "sell", 100, 110, 12)
			row := syntheticSBIReport(100)
			for _, e := range []*store.Execution{&a, &b} {
				changeSBIDetails(e, func(d map[string]any) { d["broker_reported_realized_pnl"] = 1000.0 })
				if short {
					e.Side = "buy"
					changeSBIDetails(e, func(d map[string]any) { d["lot"] = "sbi:margin-short" })
				}
			}
			if short {
				row.Transaction = "信用返済買"
			}
			fills := []store.Execution{b, a}
			for range 2 {
				got := MatchSBIMarginPnL([]SBIMarginPnLRow{row, row}, fills)
				require.NotEqual(t, got[0].ExecutionID, got[1].ExecutionID)
				for _, r := range got {
					require.Equal(t, "enrichable", r.Status)
					require.Equal(t, "equivalent", r.MatchProvenance)
					require.Equal(t, "unresolved", r.OpeningMatch)
					require.Empty(t, r.Allocations)
				}
				slices.Reverse(fills)
			}
			changed := row
			changed.Basis++
			for _, rows := range [][]SBIMarginPnLRow{{row}, {row, row, row}, {row, changed}} {
				for _, r := range MatchSBIMarginPnL(rows, fills) {
					require.Equal(t, "ambiguous_match", r.Status)
				}
			}
			// Unknown evidence can also fit a different result; it must remain a candidate.
			changeSBIDetails(&b, func(d map[string]any) { delete(d, "broker_reported_realized_pnl") })
			c := a
			c.ID = "c"
			changed = row
			pnl := 900.0
			changed.PnL = &pnl
			got := MatchSBIMarginPnL([]SBIMarginPnLRow{row, row, changed}, []store.Execution{a, b, c})
			require.Equal(t, "ambiguous_match", got[0].Status)
			require.Equal(t, "ambiguous_match", got[1].Status)
		})
	}
}

func TestSBIUnresolvedEnrichmentPreservesExecutionsAndValuation(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "synthetic.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := t.Context()
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "synthetic-350@test.invalid", PasswordHash: "x"})
	require.NoError(t, err)
	_, err = q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: "u", Name: "Synthetic", Broker: "sbi", BaseCurrency: "JPY"})
	require.NoError(t, err)
	history, err := os.ReadFile("testdata/sbi-margin-pnl-unresolved-history.csv")
	require.NoError(t, err)
	_, raw, ok, err := ReadSBITradeCSV(history)
	require.NoError(t, err)
	require.True(t, ok)
	parsed := ParseSBIRows(raw, nil, "")
	require.Empty(t, parsed.Errors)
	_, err = Commit(ctx, q, "u", "a", sql.NullString{}, parsed)
	require.NoError(t, err)
	query := store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"}
	before, err := q.ListExecutionsForAccount(ctx, query)
	require.NoError(t, err)
	dates := []time.Time{time.Date(2031, 4, 7, 16, 0, 0, 0, time.UTC), time.Date(2031, 4, 8, 16, 0, 0, 0, time.UTC)}
	snapshot := positions.Replay(before, dates)
	data, err := os.ReadFile("testdata/sbi-margin-pnl-unresolved-report.csv")
	require.NoError(t, err)
	report, ok, err := ReadSBIMarginPnLCSV(data)
	require.NoError(t, err)
	require.True(t, ok)
	matched := MatchSBIMarginPnL(report.Parsed, before)
	n, err := CommitSBIMarginPnL(ctx, q, "u", "a", matched)
	require.NoError(t, err)
	require.Equal(t, 6, n)
	after, err := q.ListExecutionsForAccount(ctx, query)
	require.NoError(t, err)
	require.Equal(t, snapshot, positions.Replay(after, dates))
	require.Len(t, after, len(before))
	for i, e := range after {
		expected := before[i]
		expected.Details = e.Details
		require.Equal(t, expected, e)
		old, current := sbiDetails(before[i]), sbiDetails(e)
		require.Equal(t, old["broker_reported_realized_pnl"], current["broker_reported_realized_pnl"])
		for _, r := range matched {
			if r.ExecutionID != e.ID {
				continue
			}
			require.Equal(t, r.Basis, current["broker_reported_close_basis"])
			require.Equal(t, r.MatchProvenance, current["sbi_report_match_provenance"])
			require.Equal(t, r.OpeningMatch, current["sbi_opening_match_provenance"])
			if r.OpeningMatch == "unresolved" {
				require.Equal(t, "null", current["sbi_opening_allocations"])
			}
		}
	}
	for range 2 {
		slices.Reverse(report.Parsed)
		repeated := MatchSBIMarginPnL(report.Parsed, after)
		n, err = CommitSBIMarginPnL(ctx, q, "u", "a", repeated)
		require.NoError(t, err)
		require.Zero(t, n)
		stable, err := q.ListExecutionsForAccount(ctx, query)
		require.NoError(t, err)
		require.Equal(t, after, stable)
	}
}
func TestSBIConditionalFIFOAllocations(t *testing.T) {
	for _, qty := range []float64{50, 100, 150, 200} {
		t.Run(fmt.Sprint(qty), func(t *testing.T) {
			a, b, c := syntheticSBIFill("uuid-z", "buy", 100, 100, 9), syntheticSBIFill("uuid-a", "buy", 100, 100, 10), syntheticSBIFill("close", "sell", qty, 110, 11)
			expected := []SBIOpeningAllocation{{[]string{a.ID}, min(qty, 100)}}
			if qty > 100 {
				expected = append(expected, SBIOpeningAllocation{[]string{b.ID}, qty - 100})
			}
			for _, fills := range [][]store.Execution{{a, b, c}, {c, b, a}, {b, c, a}} {
				got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(qty)}, fills)[0]
				require.Equal(t, "enrichable", got.Status)
				require.Equal(t, "fifo_inferred", got.OpeningMatch)
				require.Equal(t, expected, got.Allocations)
				require.Contains(t, got.Message, "SBI's actual repayment order")
				sum := 0.0
				for _, a := range got.Allocations {
					sum += a.Quantity
				}
				require.Equal(t, qty, sum)
			}
		})
	}
}
func TestSBIRejectsNonEquivalentFIFO(t *testing.T) {
	cases := []struct {
		name   string
		change func(*store.Execution, *store.Execution)
	}{
		{"price", func(a, b *store.Execution) { b.Price = 99 }},
		{"date", func(a, b *store.Execution) { b.ExecutedAt = b.ExecutedAt.AddDate(0, 0, -1) }},
		{"class", func(a, b *store.Execution) {
			changeSBIDetails(b, func(d map[string]any) { d["sbi_source"].(map[string]any)["信用区分"] = "一般信用" })
		}},
		{"fees", func(a, b *store.Execution) { b.Fees = 1 }},
		{"interest", func(a, b *store.Execution) {
			changeSBIDetails(b, func(d map[string]any) { d["sbi_source"].(map[string]any)["金利"] = "1" })
		}},
		{"missing", func(a, b *store.Execution) { changeSBIDetails(b, func(d map[string]any) { delete(d, "sbi_source") }) }},
		{"insufficient", func(a, b *store.Execution) { a.Quantity = 10; b.Quantity = 10 }},
		{"account", func(a, b *store.Execution) { a.AccountID = "other"; b.AccountID = "other" }},
		{"direction", func(a, b *store.Execution) {
			for _, e := range []*store.Execution{a, b} {
				changeSBIDetails(e, func(d map[string]any) { d["lot"] = "sbi:margin-short" })
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, b, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 150, 110, 11)
			tc.change(&a, &b)
			got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(150)}, []store.Execution{a, b, c})
			if tc.name == "account" {
				require.Equal(t, "ambiguous_match", got[0].Status)
				return
			}
			require.Equal(t, "enrichable", got[0].Status)
			require.Equal(t, "exact", got[0].MatchProvenance)
			require.Equal(t, "unresolved", got[0].OpeningMatch)
			require.Empty(t, got[0].Allocations)
		})
	}
}
func TestSBIEqualTimesRemainEquivalentGroup(t *testing.T) {
	a, b, c := syntheticSBIFill("z", "buy", 100, 100, 9), syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("close", "sell", 150, 110, 11)
	for _, dateOnly := range []bool{false, true} {
		if dateOnly {
			for _, e := range []*store.Execution{&a, &b, &c} {
				changeSBIDetails(e, func(d map[string]any) { d["source_time_precision"] = "date" })
			}
			b.ExecutedAt = b.ExecutedAt.Add(time.Microsecond)
		}
		fills := []store.Execution{a, b, c}
		for range 2 {
			got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(150)}, fills)[0]
			require.Equal(t, "fifo_inferred", got.OpeningMatch)
			require.Equal(t, []SBIOpeningAllocation{{[]string{"a", "z"}, 150}}, got.Allocations)
			slices.Reverse(fills)
		}
	}
}
func TestSBIFIFOConsumesRemaining(t *testing.T) {
	fills := []store.Execution{syntheticSBIFill("first", "buy", 100, 100, 9), syntheticSBIFill("second", "buy", 100, 100, 10), syntheticSBIFill("close1", "sell", 50, 110, 11), syntheticSBIFill("close2", "sell", 100, 110, 12)}
	rows := []SBIMarginPnLRow{syntheticSBIReport(50), syntheticSBIReport(100)}
	for range 2 {
		got := MatchSBIMarginPnL(rows, fills)
		for _, r := range got {
			require.Equal(t, "enrichable", r.Status)
			if r.Quantity == 100 {
				require.Equal(t, []SBIOpeningAllocation{{[]string{"first"}, 50}, {[]string{"second"}, 50}}, r.Allocations)
			}
		}
		slices.Reverse(rows)
		slices.Reverse(fills)
	}
}
func TestSBIDecimalsDoNotUseDisplayTolerance(t *testing.T) {
	a, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("c", "sell", 100, 110.00001, 11)
	require.Equal(t, "no_matching_execution", MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(100)}, []store.Execution{a, c})[0].Status)
	require.True(t, sbiDecimal("1,000.00", "1000"))
	require.False(t, sbiDecimal("1000.00001", "1000"))
}

func TestSBIBrokerOpeningEvidenceAndUnstableCloseTies(t *testing.T) {
	a, b, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 50, 110, 11)
	a.ExecutedAt = a.ExecutedAt.AddDate(0, 0, -1)
	row := syntheticSBIReport(50)
	row.Evidence = map[string]string{"建日": "2032/03/01", "建単価": "100.00", "信用区分": "制度信用"}
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, []store.Execution{b, c, a})[0]
	require.Equal(t, "exact", got.OpeningMatch)
	require.Equal(t, []SBIOpeningAllocation{{[]string{"a"}, 50}}, got.Allocations)
	// No row/ID ordering may decide which equal-time repayment overdraws a lot.
	a = syntheticSBIFill("open", "buy", 100, 100, 9)
	c = syntheticSBIFill("c", "sell", 80, 110, 11)
	d := syntheticSBIFill("d", "sell", 80, 120, 11)
	r1 := syntheticSBIReport(80)
	r2 := r1
	r2.Price = 120
	pnl := 1600.0
	r2.PnL = &pnl
	fills := []store.Execution{a, c, d}
	rows := []SBIMarginPnLRow{r1, r2}
	for range 2 {
		for _, r := range MatchSBIMarginPnL(rows, fills) {
			require.Equal(t, "enrichable", r.Status)
			require.Equal(t, "exact", r.MatchProvenance)
			require.Equal(t, "unresolved", r.OpeningMatch)
			require.Empty(t, r.Allocations)
		}
		slices.Reverse(fills)
		slices.Reverse(rows)
	}
}

func TestSBIRawChargesAndConflictingDuplicateReports(t *testing.T) {
	a, b, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 150, 110, 11)
	changeSBIDetails(&b, func(d map[string]any) { d["sbi_source"].(map[string]any)["手数料/諸経費等"] = "1" })
	matched := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(150)}, []store.Execution{a, b, c})[0]
	require.Equal(t, "enrichable", matched.Status)
	require.Equal(t, "unresolved", matched.OpeningMatch)
	require.Empty(t, matched.Allocations)
	c.Quantity = 50
	r1 := syntheticSBIReport(50)
	r2 := r1
	r1.Evidence = map[string]string{"金利": "0"}
	r2.Evidence = map[string]string{"金利": "1"}
	changeSBIDetails(&c, func(d map[string]any) { delete(d, "sbi_source") })
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{r1, r2}, []store.Execution{a, c})
	for _, r := range got {
		require.Equal(t, "conflict", r.Status)
	}
	// A non-SBI close remains outside this resolver even in a margin-labelled account.
	changeSBIDetails(&c, func(d map[string]any) { d["lot"] = "ibkr:margin-long" })
	require.Equal(t, "no_matching_execution", MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(50)}, []store.Execution{a, c})[0].Status)
}

func TestSBIOriginalDecimalEvidenceIsNotRounded(t *testing.T) {
	a, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("c", "sell", 100, 110, 11)
	changeSBIDetails(&c, func(d map[string]any) { d["sbi_source"].(map[string]any)["約定単価"] = "110.000000000000000001" })
	r := syntheticSBIReport(100)
	r.Evidence = map[string]string{"約定単価": "110.000000000000000002"}
	require.Equal(t, "conflict", MatchSBIMarginPnL([]SBIMarginPnLRow{r}, []store.Execution{a, c})[0].Status)
}

func TestSBIFIFOPersistenceConservesValuationAndSettlement(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "synthetic.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.New(conn)
	ctx := t.Context()
	_, err = q.CreateUser(ctx, store.CreateUserParams{ID: "u", Email: "synthetic@test.invalid", PasswordHash: "x"})
	require.NoError(t, err)
	_, err = q.CreateAccount(ctx, store.CreateAccountParams{ID: "a", UserID: "u", Name: "Synthetic", Broker: "sbi", BaseCurrency: "JPY"})
	require.NoError(t, err)
	fills := []store.Execution{syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 150, 110, 11)}
	parsed := ParseResult{Format: "executions"}
	pnl := 1497.0
	for _, f := range fills {
		source := sbiSource(f)
		source["手数料/諸経費等"] = "1"
		pe := ParsedExecution{DedupKey: f.ID, Symbol: f.Symbol, InstrumentType: "stock", Side: f.Side, PositionType: PositionMarginLong, PositionEffect: PositionIncrease, Quantity: f.Quantity, Price: f.Price, Fees: 1, ExecutedAt: f.ExecutedAt, Multiplier: 1, LotKey: "sbi:margin-long", SBIFields: source}
		if f.Side == "sell" {
			pe.PositionEffect = PositionReduce
			pe.ReportedRealizedPnl = &pnl
		}
		parsed.Executions = append(parsed.Executions, pe)
	}
	_, err = Commit(ctx, q, "u", "a", sql.NullString{}, parsed)
	require.NoError(t, err)
	query := store.ListExecutionsForAccountParams{UserID: "u", AccountID: "a"}
	before, err := q.ListExecutionsForAccount(ctx, query)
	require.NoError(t, err)
	dates := []time.Time{time.Date(2032, 3, 2, 16, 0, 0, 0, time.UTC)}
	snapshot := positions.Replay(before, dates)
	row := syntheticSBIReport(150)
	row.PnL = &pnl
	matched := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, before)
	require.Equal(t, "fifo_inferred", matched[0].OpeningMatch)
	tampered := append([]SBIMarginPnLRow(nil), matched...)
	tampered[0].Allocations = append([]SBIOpeningAllocation(nil), matched[0].Allocations...)
	tampered[0].Allocations[0].Quantity++
	_, err = CommitSBIMarginPnL(ctx, q, "u", "a", tampered)
	require.ErrorContains(t, err, "preview again")
	unchanged, err := q.ListExecutionsForAccount(ctx, query)
	require.NoError(t, err)
	require.Equal(t, before, unchanged)
	n, err := CommitSBIMarginPnL(ctx, q, "u", "a", matched)
	require.NoError(t, err)
	require.Equal(t, 1, n)
	after, err := q.ListExecutionsForAccount(ctx, query)
	require.NoError(t, err)
	require.Equal(t, snapshot, positions.Replay(after, dates))
	// Historical valuation replay retains its existing allocated-fee model.
	require.Equal(t, 1497.5, snapshot[0].Accounts[0].RealizedPnL)
	require.Equal(t, 100.0, snapshot[0].Accounts[0].Positions[0].AverageCost)
	require.Equal(t, 50.0, snapshot[0].Accounts[0].Positions[0].Quantity)
	result, err := trades.NewService(q).SBIAccounting(ctx, after)
	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Equal(t, 1497.0, result[0].RealizedCloses[0].Pnl)
	require.Equal(t, 50.0, result[0].RealizedCloses[0].RemainingQty)
	// Persisted provenance remains inferred after repeat matching; no writes on replay.
	repeated := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, after)
	require.Equal(t, "already_enriched", repeated[0].Status)
	require.Equal(t, "fifo_inferred", repeated[0].OpeningMatch)
	n, err = CommitSBIMarginPnL(ctx, q, "u", "a", repeated)
	require.NoError(t, err)
	require.Zero(t, n)
}

func TestSBINewCycleAfterUnresolvedFlatHistory(t *testing.T) {
	a, b, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 200, 111, 11)
	for _, e := range []*store.Execution{&a, &b} {
		changeSBIDetails(e, func(d map[string]any) { delete(d, "sbi_source") })
	}
	changeSBIDetails(&c, func(d map[string]any) { d["broker_reported_close_basis"] = 100.0 })
	d, e, f := syntheticSBIFill("d", "buy", 100, 100, 12), syntheticSBIFill("e", "buy", 100, 100, 13), syntheticSBIFill("f", "sell", 150, 110, 14)
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(150)}, []store.Execution{a, b, c, d, e, f})[0]
	require.Equal(t, "enrichable", got.Status)
	require.Equal(t, "fifo_inferred", got.OpeningMatch)
	require.Equal(t, []SBIOpeningAllocation{{[]string{"d"}, 100}, {[]string{"e"}, 50}}, got.Allocations)
}

func TestSBIStoredOpeningTermsOverrideFIFO(t *testing.T) {
	a, b, c := syntheticSBIFill("older", "buy", 100, 100, 9), syntheticSBIFill("newer", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 50, 110, 11)
	changeSBIDetails(&a, func(d map[string]any) { d["sbi_source"].(map[string]any)["預り"] = "一般" })
	for _, e := range []*store.Execution{&b, &c} {
		changeSBIDetails(e, func(d map[string]any) { d["sbi_source"].(map[string]any)["預り"] = "特定" })
	}
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(50)}, []store.Execution{a, b, c})[0]
	require.Equal(t, "exact", got.OpeningMatch)
	require.Equal(t, []SBIOpeningAllocation{{[]string{"newer"}, 50}}, got.Allocations)
}

func TestSBIBrokerOpeningPricePrecedesAverageSettlementBasis(t *testing.T) {
	a, c := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("c", "sell", 50, 110, 11)
	row := syntheticSBIReport(50)
	row.Basis = 100.5
	row.Evidence = map[string]string{"建日": "2032/03/02", "建単価": "100", "信用区分": "制度信用"}
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, []store.Execution{a, c})[0]
	require.Equal(t, "enrichable", got.Status)
	require.Equal(t, "exact", got.OpeningMatch)
	require.Equal(t, []SBIOpeningAllocation{{[]string{"a"}, 50}}, got.Allocations)
}

func TestSBIInferredResidualCannotBecomeBrokerExact(t *testing.T) {
	fills := []store.Execution{syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 100, 10), syntheticSBIFill("c", "sell", 100, 110, 11), syntheticSBIFill("d", "sell", 50, 110, 12)}
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{syntheticSBIReport(100), syntheticSBIReport(50)}, fills)
	for _, r := range got {
		require.Equal(t, "enrichable", r.Status)
		require.Equal(t, "fifo_inferred", r.OpeningMatch)
	}
	require.Equal(t, []SBIOpeningAllocation{{[]string{"b"}, 50}}, got[1].Allocations)
}

func TestSBIBrokerDateExcludesOtherDaysBeforeAverageBasisProof(t *testing.T) {
	a, b, c, d := syntheticSBIFill("a", "buy", 100, 100, 9), syntheticSBIFill("b", "buy", 100, 90, 10), syntheticSBIFill("c", "buy", 100, 110, 11), syntheticSBIFill("d", "sell", 50, 110, 12)
	a.ExecutedAt = a.ExecutedAt.AddDate(0, 0, -1)
	row := syntheticSBIReport(50)
	row.Evidence = map[string]string{"建日": "2032/03/01"}
	got := MatchSBIMarginPnL([]SBIMarginPnLRow{row}, []store.Execution{a, b, c, d})[0]
	require.Equal(t, "enrichable", got.Status)
	require.Equal(t, "exact", got.OpeningMatch)
	require.Equal(t, []SBIOpeningAllocation{{[]string{"a"}, 50}}, got.Allocations)
}
