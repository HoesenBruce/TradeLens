package marketdata

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

var commonSplitRatios = []float64{2, 3, 4, 5, 6, 10}

// CorporateActionCandidate is evidence of a possible unsupported boundary.
// Candidates are never authoritative: imported broker records remain unchanged.
type CorporateActionCandidate struct {
	Instrument       string   `json:"instrument"`
	EffectiveDate    string   `json:"effective_date"`
	CandidateType    string   `json:"candidate_type"`
	SuspectedRatio   float64  `json:"suspected_ratio,omitempty"`
	Confidence       float64  `json:"confidence"`
	Evidence         []string `json:"evidence"`
	Source           string   `json:"source"`
	AdjustmentStatus string   `json:"adjustment_status"`
	Status           string   `json:"status"`
}

// DetectCorporateActions loads daily bars through the shared market-data service.
func (s *Service) DetectCorporateActions(ctx context.Context, req Request) ([]CorporateActionCandidate, error) {
	req.Interval = "D"
	response, err := s.GetBars(ctx, req)
	if err != nil {
		return nil, err
	}
	return FindCorporateActionCandidates(response), nil
}

// FindCorporateActionCandidates detects large day-to-day scaling discontinuities.
// It reports reviewable candidates; it never rewrites prices or executions.
func FindCorporateActionCandidates(response Response) []CorporateActionCandidate {
	bars := append([]Bar(nil), response.Bars...)
	sort.SliceStable(bars, func(i, j int) bool { return bars[i].Time < bars[j].Time })

	candidates := make([]CorporateActionCandidate, 0)
	for _, bar := range bars {
		if bar.SplitRatio <= 0 || bar.SplitRatio == 1 {
			continue
		}
		candidateType, ratio := "stock_split", bar.SplitRatio
		if ratio < 1 {
			candidateType, ratio = "reverse_stock_split", 1/ratio
		}
		candidates = append(candidates, CorporateActionCandidate{
			Instrument: response.Instrument, EffectiveDate: marketDate(bar),
			CandidateType: candidateType, SuspectedRatio: round(ratio, 3), Confidence: 0.95,
			Evidence: []string{fmt.Sprintf("%s reported %.3g:1 split", response.Source, ratio)},
			Source:   response.Source, AdjustmentStatus: response.AdjustmentStatus, Status: "unconfirmed",
		})
	}
	for i := 1; i < len(bars); i++ {
		previous, current := bars[i-1], bars[i]
		if current.SplitRatio > 0 || !validPrices(previous) || !validPrices(current) {
			continue
		}

		ratio := previous.Close / current.Open
		discontinuity := math.Max(ratio, 1/ratio)
		if discontinuity < 1.5 {
			continue
		}

		commonRatio, common := nearestCommonRatio(discontinuity)
		consistent := scalesConsistently(previous, current, discontinuity)
		candidateType := "other_corporate_action"
		if response.AdjustmentStatus != "" && response.AdjustmentStatus != "unadjusted" {
			candidateType = "adjustment_mismatch"
		} else if common && consistent {
			if ratio >= 1 {
				candidateType = "stock_split"
			} else {
				candidateType = "reverse_stock_split"
			}
		}

		confidence := 0.45
		evidence := []string{fmt.Sprintf("previous close/current open ratio %.3f", ratio)}
		if common {
			confidence += 0.25
			discontinuity = commonRatio
			evidence = append(evidence, fmt.Sprintf("ratio is near common %.0f:1 split", commonRatio))
		}
		if consistent {
			confidence += 0.2
			evidence = append(evidence, "OHLC values scale consistently")
		}
		if candidateType == "adjustment_mismatch" {
			confidence = math.Min(confidence, 0.6)
			evidence = append(evidence, "provider adjustment status is "+response.AdjustmentStatus)
		}

		candidates = append(candidates, CorporateActionCandidate{
			Instrument: response.Instrument, EffectiveDate: marketDate(current),
			CandidateType: candidateType, SuspectedRatio: round(discontinuity, 3),
			Confidence: round(math.Min(confidence, 0.95), 2), Evidence: evidence,
			Source: response.Source, AdjustmentStatus: response.AdjustmentStatus, Status: "unconfirmed",
		})
	}
	return candidates
}

// CorporateActionWarnings returns boundaries a downstream reconstruction must
// surface as incomplete/unsupported instead of silently changing quantities.
func CorporateActionWarnings(candidates []CorporateActionCandidate, from, to time.Time) []string {
	warnings := make([]string, 0)
	for _, candidate := range candidates {
		date, err := time.Parse("2006-01-02", candidate.EffectiveDate)
		if err != nil || date.Before(dateOnly(from)) || date.After(dateOnly(to)) || candidate.Status == "rejected" {
			continue
		}
		warnings = append(warnings, fmt.Sprintf(
			"unconfirmed %s candidate for %s on %s (ratio %.3g)",
			strings.ReplaceAll(candidate.CandidateType, "_", " "), candidate.Instrument,
			candidate.EffectiveDate, candidate.SuspectedRatio,
		))
	}
	return warnings
}

func validPrices(bar Bar) bool {
	return bar.Open > 0 && bar.High > 0 && bar.Low > 0 && bar.Close > 0
}

func nearestCommonRatio(ratio float64) (float64, bool) {
	nearest := commonSplitRatios[0]
	for _, candidate := range commonSplitRatios[1:] {
		if math.Abs(candidate-ratio) < math.Abs(nearest-ratio) {
			nearest = candidate
		}
	}
	return nearest, math.Abs(nearest-ratio)/nearest <= 0.08
}

func scalesConsistently(previous, current Bar, expected float64) bool {
	ratios := []float64{
		previous.Open / current.Open,
		previous.High / current.High,
		previous.Low / current.Low,
		previous.Close / current.Close,
	}
	for _, ratio := range ratios {
		if ratio < 1 {
			ratio = 1 / ratio
		}
		if math.Abs(ratio-expected)/expected > 0.12 {
			return false
		}
	}
	return true
}

func marketDate(bar Bar) string {
	if bar.MarketDate != "" {
		return bar.MarketDate
	}
	return time.Unix(bar.Time, 0).UTC().Format("2006-01-02")
}

func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}
