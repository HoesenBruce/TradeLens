package importer

import (
	"database/sql"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/store"
)

func TestReprocessedExecutionRetainsSBISourceEvidence(t *testing.T) {
	pe := ParsedExecution{DedupKey: "broker-fill", Multiplier: 1, SBIFields: map[string]string{"信用区分": "制度信用", "預り": "特定"}}
	first, err := executionParams(pe, "fill", "user", "account", sql.NullString{}, "hash", 1, 0)
	require.NoError(t, err)
	var details map[string]string
	require.NoError(t, json.Unmarshal([]byte(first.Details.String), &details))
	var source map[string]string
	require.NoError(t, json.Unmarshal([]byte(details["sbi_source"]), &source))
	require.Equal(t, pe.SBIFields, source)

	old := store.Execution{ID: first.ID, UserID: first.UserID, AccountID: first.AccountID, DedupHash: first.DedupHash, Details: first.Details}
	replayed, err := ReprocessedExecution(t.Context(), nil, old, pe)
	require.NoError(t, err)
	require.Equal(t, first.Details, replayed.Details)
	pe.SBIFields["預り"] = "一般"
	replayed, err = ReprocessedExecution(t.Context(), nil, old, pe)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal([]byte(replayed.Details.String), &details))
	require.NoError(t, json.Unmarshal([]byte(details["sbi_source"]), &source))
	require.Equal(t, pe.SBIFields, source)
}
