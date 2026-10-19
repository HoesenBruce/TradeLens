package importer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tradermemos/api/internal/store"
)

// ReprocessedExecution uses the same serialization as first import, retaining identity.
func ReprocessedExecution(ctx context.Context, q store.Querier, old store.Execution, pe ParsedExecution) (store.ReplaceImportedExecutionParams, error) {
	details := map[string]json.RawMessage{}
	if old.Details.Valid {
		if err := json.Unmarshal([]byte(old.Details.String), &details); err != nil {
			return store.ReplaceImportedExecutionParams{}, err
		}
	}
	var occurrenceText string
	_ = json.Unmarshal(details["occ"], &occurrenceText)
	occurrence := DedupOccurrence(map[string]string{"occ": occurrenceText})
	text := func(key string) string { var value string; _ = json.Unmarshal(details[key], &value); return value }
	hash := old.DedupHash
	if pe.DedupKey == "" {
		hash = DedupHashOccurrence(dedupSymbol(pe), pe.Side, pe.Quantity, pe.Price, pe.ExecutedAt, occurrence)
	}
	// Random journal/round-trip lots and broker event IDs already identify existing records.
	if pe.LotKey != "" && !strings.HasPrefix(pe.LotKey, "sbi:") {
		pe.LotKey = text("lot")
	}
	if pe.ConversionID != "" {
		pe.ConversionID = text("conversion_id")
	}
	if pe.SettlementID != "" {
		pe.SettlementID = text("settlement_id")
	}
	params, err := executionParams(pe, old.ID, old.UserID, old.AccountID, old.ImportBatchID, hash, resolveMultiplierCached(ctx, q, map[string]float64{}, pe), occurrence)
	if err != nil {
		return store.ReplaceImportedExecutionParams{}, err
	}
	// Preserve supplemental broker evidence and derived fields not supplied by this source row.
	updated := map[string]json.RawMessage{}
	if params.Details.Valid {
		if err := json.Unmarshal([]byte(params.Details.String), &updated); err != nil {
			return store.ReplaceImportedExecutionParams{}, fmt.Errorf("execution details: %w", err)
		}
	}
	for _, key := range []string{"option_right", "strike", "expiry", "stock_name", "position_type", "position_effect", "event_type", "conversion_type", "conversion_id", "settlement_type", "settlement_id", "source_time_precision"} {
		delete(details, key)
	}
	for key, value := range updated {
		details[key] = value
	}
	if len(details) > 0 {
		data, err := json.Marshal(details)
		if err != nil {
			return store.ReplaceImportedExecutionParams{}, err
		}
		params.Details.String = string(data)
		params.Details.Valid = true
	}
	return store.ReplaceImportedExecutionParams{
		ID: old.ID, UserID: old.UserID, AccountID: old.AccountID, ExternalID: params.ExternalID,
		Symbol: params.Symbol, InstrumentType: params.InstrumentType, Side: params.Side,
		Quantity: params.Quantity, Price: params.Price, Fees: params.Fees, Commission: params.Commission,
		ExecutedAt: params.ExecutedAt, Multiplier: params.Multiplier, Details: params.Details, DedupHash: params.DedupHash,
	}, nil
}
