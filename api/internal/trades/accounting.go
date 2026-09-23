package trades

import "time"

// RealizedClose is separate from Trade.NetPnl: the default trade remains open
// after a partial close, while a broker may report a realized result for it.
type RealizedClose struct {
	ExecutionID        string
	Date               time.Time
	Pnl                float64
	Fees               float64 // realized fees; conversion costs are capitalized instead
	RemainingQty       float64
	RemainingCostBasis float64
	Source             string
}

type AccountingResult struct {
	Trades              []Trade
	RealizedCloses      []RealizedClose
	UnavailableCloseIDs []string
}

type AccountingStrategy func([]Execution) AccountingResult

// Account selects only explicit broker/position buckets. An unregistered
// bucket, including every existing non-SBI source, keeps the upstream engine.
func Account(fills []Execution, strategies map[string]AccountingStrategy) AccountingResult {
	if len(fills) > 0 {
		lot := fills[0].LotKey
		if lot != "" && strategies != nil {
			if strategy := strategies[lot]; strategy != nil {
				return strategy(fills)
			}
		}
	}
	return AccountingResult{Trades: Group(fills)}
}
