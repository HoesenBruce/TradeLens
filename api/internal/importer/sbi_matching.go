package importer

import (
	"encoding/json"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tradermemos/api/internal/store"
)

// Multiple IDs denote an equivalent group, never an SBI-designated physical lot.
type SBIOpeningAllocation struct {
	ExecutionIDs []string `json:"execution_ids"`
	Quantity     float64  `json:"quantity"`
}

func isSBICredit(r SBIMarginPnLRow) bool {
	return r.Transaction == "信用返済売" || r.Transaction == "信用返済買"
}

// Preserve supplied terms and explicit zeroes; missing values stay missing.
func sbiEvidence(row map[string]string) map[string]string {
	out := map[string]string{}
	for _, key := range []string{"建日", "建単価", "信用区分", "信用種類", "信用種別", "信用取引区分", "口座", "預り", "預り区分", "建市場", "管理費", "権利処理等手数料", "名義書換料", "市場", "返済期限", "金利", "貸株料", "その他諸経費", "手数料/諸経費等", "手数料", "消費税", "受渡日"} {
		if v := row[key]; v != "" && v != "--" && v != "-" {
			out[key] = v
		}
	}
	for key, raw := range map[string]string{
		"約定単価":   firstValue(row, "約定単価", "単価"),
		"約定数量":   firstValue(row, "約定数量", "数量"),
		"平均取得価額": row["平均取得価額"],
	} {
		if raw != "" && raw != "--" && raw != "-" {
			out[key] = raw
		}
	}
	return out
}
func sbiDetails(e store.Execution) map[string]any {
	d := map[string]any{}
	_ = json.Unmarshal([]byte(e.Details.String), &d)
	return d
}
func sbiSource(e store.Execution) map[string]string {
	var d struct {
		Source json.RawMessage `json:"sbi_source"`
	}
	_ = json.Unmarshal([]byte(e.Details.String), &d)
	var source map[string]string
	var encoded string
	if json.Unmarshal(d.Source, &encoded) == nil {
		_ = json.Unmarshal([]byte(encoded), &source)
	} else {
		_ = json.Unmarshal(d.Source, &source)
	}
	return source
}

func sbiDecimalCompare(a, b string) (int, bool) {
	x, ok := new(big.Rat).SetString(strings.ReplaceAll(a, ",", ""))
	if !ok {
		return 0, false
	}
	y, ok := new(big.Rat).SetString(strings.ReplaceAll(b, ",", ""))
	if !ok {
		return 0, false
	}
	return x.Cmp(y), true
}
func sbiDecimal(a, b string) bool { cmp, ok := sbiDecimalCompare(a, b); return ok && cmp == 0 }

func sbiTermEqual(key, a, b string) bool {
	switch key {
	case "建日", "受渡日", "返済期限":
		x, xe := sbiPnLDate(a)
		y, ye := sbiPnLDate(b)
		return xe == nil && ye == nil && x == y
	case "約定単価", "約定数量", "平均取得価額", "建単価", "管理費", "権利処理等手数料", "名義書換料", "金利", "貸株料", "その他諸経費", "手数料/諸経費等", "手数料", "消費税":
		return sbiDecimal(a, b)
	default:
		return a == b
	}
}
func sbiConsistent(r SBIMarginPnLRow, e store.Execution) bool {
	d := sbiDetails(e)
	if n, ok := d["broker_reported_close_basis"].(float64); ok && n != r.Basis {
		return false
	}
	if n, ok := d["broker_reported_realized_pnl"].(float64); ok && (r.PnL == nil || n != *r.PnL) {
		return false
	}
	source := sbiSource(e)
	for key, value := range r.Evidence {
		if stored := source[key]; stored != "" && !sbiTermEqual(key, value, stored) {
			return false
		}
	}
	return true
}

var sbiZone = time.FixedZone("JST", 9*3600)

func sbiDay(e store.Execution) string { return e.ExecutedAt.In(sbiZone).Format(time.DateOnly) }
func sbiOriginalTime(e store.Execution) time.Time {
	if sbiDetails(e)["source_time_precision"] == "date" {
		day, _ := time.Parse(time.DateOnly, sbiDay(e))
		return day
	}
	if at, err := time.Parse(time.RFC3339Nano, sbiSource(e)["original_executed_at"]); err == nil {
		return at
	}
	return e.ExecutedAt
}
func sbiUnitChargeEqual(a, b string, aq, bq float64) bool {
	x, ok := new(big.Rat).SetString(strings.ReplaceAll(a, ",", ""))
	if !ok {
		return false
	}
	y, ok := new(big.Rat).SetString(strings.ReplaceAll(b, ",", ""))
	if !ok {
		return false
	}
	qx, ok := new(big.Rat).SetString(strconv.FormatFloat(aq, 'f', -1, 64))
	if !ok || qx.Sign() <= 0 {
		return false
	}
	qy, ok := new(big.Rat).SetString(strconv.FormatFloat(bq, 'f', -1, 64))
	if !ok || qy.Sign() <= 0 {
		return false
	}
	return x.Quo(x, qx).Cmp(y.Quo(y, qy)) == 0
}

func sbiOpeningConsistent(e store.Execution) bool {
	source := sbiSource(e)
	if raw := source["約定数量"]; raw != "" && !sbiDecimal(raw, strconv.FormatFloat(e.Quantity, 'f', -1, 64)) {
		return false
	}
	if date := source["建日"]; date != "" {
		day, err := sbiPnLDate(date)
		if err != nil || day != sbiDay(e) {
			return false
		}
	}
	if price := source["建単価"]; price != "" && !sbiDecimal(price, sbiRawPrice(e)) {
		return false
	}
	return true
}

func sbiRawPrice(e store.Execution) string {
	if raw := sbiSource(e)["約定単価"]; raw != "" {
		return raw
	}
	return strconv.FormatFloat(e.Price, 'f', -1, 64)
}
func sbiRawBasis(r SBIMarginPnLRow) string {
	if raw := r.Evidence["平均取得価額"]; raw != "" {
		return raw
	}
	return strconv.FormatFloat(r.Basis, 'f', -1, 64)
}
func sbiOpeningTermsMatch(r SBIMarginPnLRow, e store.Execution) bool {
	if date := r.Evidence["建日"]; date != "" {
		day, err := sbiPnLDate(date)
		if err != nil || day != sbiDay(e) {
			return false
		}
	}
	source := sbiSource(e)
	for _, key := range []string{"信用区分", "信用種類", "信用種別", "信用取引区分", "口座", "預り", "預り区分", "建市場", "返済期限"} {
		if expected := r.Evidence[key]; expected != "" && source[key] != "" && !sbiTermEqual(key, expected, source[key]) {
			return false
		}
	}
	return true
}

func sbiEquivalent(a, b store.Execution) bool {
	if !sbiOpeningConsistent(a) || !sbiOpeningConsistent(b) || sbiDay(a) != sbiDay(b) || a.Price != b.Price || !sbiDecimal(sbiRawPrice(a), sbiRawPrice(b)) || a.Fees/a.Quantity != b.Fees/b.Quantity || a.Commission/a.Quantity != b.Commission/b.Quantity {
		return false
	}
	x, y := sbiSource(a), sbiSource(b)
	for _, key := range []string{"信用区分", "手数料/諸経費等", "金利", "貸株料", "その他諸経費"} {
		if x[key] == "" || y[key] == "" {
			return false
		}
	}
	for key, value := range x {
		if key == "original_executed_at" || key == "original_execution_date" || key == "建日" || key == "建単価" || key == "約定数量" {
			continue
		}
		switch key {
		case "管理費", "権利処理等手数料", "名義書換料", "金利", "貸株料", "その他諸経費", "手数料/諸経費等", "手数料", "消費税":
			if !sbiUnitChargeEqual(value, y[key], a.Quantity, b.Quantity) {
				return false
			}
			continue
		}
		if !sbiTermEqual(key, value, y[key]) {
			return false
		}
	}
	for key := range y {
		if key != "original_executed_at" && key != "original_execution_date" && key != "建日" && key != "建単価" && x[key] == "" {
			return false
		}
	}
	return true
}

type sbiOpen struct {
	execution store.Execution
	ids       []string
	remaining float64
}

// Opening allocations are provenance only: existing accounting/valuation and
// review groups keep using their established SBI strategies and original fills.
func matchSBIOpenings(rows []SBIMarginPnLRow, executions []store.Execution) {
	byID := map[string][]int{}
	for i := range rows {
		r := &rows[i]
		if !isSBICredit(*r) {
			continue
		}
		r.OpeningMatch = "unresolved"
		if r.Status != "enrichable" && r.Status != "already_enriched" {
			r.MatchProvenance = "unresolved"
			continue
		}
		byID[r.ExecutionID] = append(byID[r.ExecutionID], i)
	}
	groups := map[string][]store.Execution{}
	for _, e := range executions {
		d := sbiDetails(e)
		lot, _ := d["lot"].(string)
		if (lot != "sbi:margin-long" && lot != "sbi:margin-short") || e.InstrumentType != "stock" || e.Multiplier != 1 {
			continue
		}
		key := e.UserID + "|" + e.AccountID + "|" + e.Symbol + "|" + lot
		groups[key] = append(groups[key], e)
	}
	for _, events := range groups {
		// A date-only day has no trustworthy intraday close sequence.
		uncertain := map[string]bool{}
		for _, e := range events {
			if sbiDetails(e)["source_time_precision"] == "date" {
				uncertain[sbiDay(e)] = true
			}
		}
		sort.SliceStable(events, func(i, j int) bool {
			a, b := events[i], events[j]
			if sbiDay(a) != sbiDay(b) {
				return sbiDay(a) < sbiDay(b)
			}
			if uncertain[sbiDay(a)] {
				return sbiDetails(a)["position_effect"] == "increase" && sbiDetails(b)["position_effect"] != "increase"
			}
			return sbiOriginalTime(a).Before(sbiOriginalTime(b))
		})
		closeKey := func(e store.Execution) string {
			if uncertain[sbiDay(e)] {
				return sbiDay(e)
			}
			return sbiOriginalTime(e).Format(time.RFC3339Nano)
		}
		closeCounts := map[string]int{}
		demands := map[string]map[float64]float64{}
		missingBasis := map[string]bool{}
		for _, e := range events {
			if sbiDetails(e)["position_effect"] != "reduce" {
				continue
			}
			key := closeKey(e)
			basis, _ := sbiDetails(e)["broker_reported_close_basis"].(float64)
			if indices := byID[e.ID]; len(indices) > 0 {
				basis = rows[indices[0]].Basis
			}
			closeCounts[key]++
			if demands[key] == nil {
				demands[key] = map[float64]float64{}
			}
			demands[key][basis] += e.Quantity
			missingBasis[key] = missingBasis[key] || basis <= 0
		}
		var opens []sbiOpen
		inventory := 0.0
		closedInGroup := map[string]map[float64]float64{}
		unknown, inferredInventory := false, false
		for _, close := range events {
			d := sbiDetails(close)
			wantSide := "buy"
			if d["lot"] == "sbi:margin-short" {
				wantSide = "sell"
			}
			if d["position_effect"] == "increase" && close.Side == wantSide {
				if close.Quantity > 0 {
					inventory += close.Quantity
					merged := false
					for i := range opens {
						if sbiOriginalTime(opens[i].execution).Equal(sbiOriginalTime(close)) && sbiEquivalent(opens[i].execution, close) {
							opens[i].ids = append(opens[i].ids, close.ID)
							sort.Strings(opens[i].ids)
							opens[i].remaining += close.Quantity
							merged = true
							break
						}
					}
					if !merged {
						opens = append(opens, sbiOpen{close, []string{close.ID}, close.Quantity})
					}
				}
				continue
			}
			if d["position_effect"] != "reduce" || close.Side == wantSide {
				continue
			}
			indices := byID[close.ID]
			var r SBIMarginPnLRow
			if len(indices) > 0 {
				r = rows[indices[0]]
			} else {
				r.Basis, _ = d["broker_reported_close_basis"].(float64)
				r.Evidence = sbiSource(close)
			}
			evidence := sbiSource(close)
			if evidence == nil {
				evidence = map[string]string{}
			}
			for key, value := range r.Evidence {
				evidence[key] = value
			}
			r.Evidence = evidence
			tied := closeCounts[closeKey(close)] > 1
			openingPrice := sbiRawBasis(r)
			if price := r.Evidence["建単価"]; price != "" {
				openingPrice = price
			}
			var eligible []int
			available := 0.0
			for i, o := range opens {
				if o.remaining <= 0 {
					continue
				}
				available += o.remaining
				if !sbiOpeningConsistent(o.execution) || !sbiOpeningTermsMatch(r, o.execution) || !sbiDecimal(sbiRawPrice(o.execution), openingPrice) {
					continue
				}
				if price := r.Evidence["建単価"]; price != "" && !sbiDecimal(price, sbiRawPrice(o.execution)) {
					continue
				}
				eligible = append(eligible, i)
			}
			resolved := !unknown && len(eligible) > 0
			provenance := "exact"
			designated := !inferredInventory && len(eligible) == 1 && len(opens[eligible[0]].ids) == 1
			// An average basis identifies a unique price only at an endpoint;
			// prices on both sides could produce the same weighted average.
			lower, higher := false, false
			for _, o := range opens {
				if o.remaining <= 0 || !sbiOpeningTermsMatch(r, o.execution) {
					continue
				}
				cmp, valid := sbiDecimalCompare(sbiRawPrice(o.execution), openingPrice)
				if !valid {
					designated, resolved = false, false
				}
				lower = lower || cmp < 0
				higher = higher || cmp > 0
			}
			if lower && higher && r.Evidence["建単価"] == "" {
				designated = false
			}
			if tied && !designated {
				resolved = false
			}
			if !designated {
				provenance = "fifo_inferred"
				for _, o := range opens {
					if o.remaining > 0 && (len(eligible) == 0 || !sbiEquivalent(opens[eligible[0]].execution, o.execution)) {
						resolved = false
						break
					}
				}
			}
			quantity := close.Quantity
			eligibleQty := 0.0
			for _, i := range eligible {
				eligibleQty += opens[i].remaining
			}
			if quantity <= 0 || eligibleQty < quantity || (tied && (missingBasis[closeKey(close)] || insufficientAtCloseGroup(opens, eligible, closeKey(close), r.Basis, demands[closeKey(close)][r.Basis], closedInGroup))) {
				resolved = false
			}
			var allocations []SBIOpeningAllocation
			if resolved {
				inferredInventory = inferredInventory || provenance == "fifo_inferred"
				sort.SliceStable(eligible, func(i, j int) bool {
					return sbiOriginalTime(opens[eligible[i]].execution).Before(sbiOriginalTime(opens[eligible[j]].execution))
				})
				for _, i := range eligible {
					if quantity <= 0 {
						break
					}
					take := min(quantity, opens[i].remaining)
					allocations = append(allocations, SBIOpeningAllocation{append([]string(nil), opens[i].ids...), take})
					quantity -= take
					opens[i].remaining -= take
				}
				key := closeKey(close)
				if closedInGroup[key] == nil {
					closedInGroup[key] = map[float64]float64{}
				}
				closedInGroup[key][r.Basis] += close.Quantity
			} else {
				unknown = true
			}
			for _, i := range indices {
				row := &rows[i]
				if resolved {
					row.OpeningMatch, row.Allocations = provenance, allocations
					if provenance == "fifo_inferred" {
						row.Message = "fifo_inferred: TradeLens equivalent-lot allocation; SBI's actual repayment order and lot-level holding duration are unknown."
					}
				} else {
					// A broker-reported close result does not require a proven opening lot.
					row.Message = "Opening allocation unresolved: broker-reported close result retained; opening lots and lot-level holding duration are unknown."
					if available < close.Quantity {
						row.Message = "Insufficient supported opening quantity: broker-reported close result retained; missing holdings and opening lots are not inferred."
					}
				}
			}
			inventory -= close.Quantity
			if inventory == 0 {
				opens, unknown, inferredInventory = nil, false, false
			}

		}
	}
}

func insufficientAtCloseGroup(opens []sbiOpen, eligible []int, key string, basis, demand float64, consumed map[string]map[float64]float64) bool {
	if len(eligible) == 0 {
		return true
	}
	available := consumed[key][basis]
	for _, i := range eligible {
		available += opens[i].remaining
	}
	return available < demand
}
