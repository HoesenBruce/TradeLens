package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportOccurrenceEditAndReimport(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "occ-edit@x.com")
	acc := accountID(t, s, tok)
	csv := "symbol,side,quantity,price,executed_at\nAAPL260821C00120000,buy,1,10,2026-01-01T10:00:00Z\nAAPL260821C00120000,buy,1,10,2026-01-01T10:00:00Z\n"
	fields := map[string]string{"account_id": acc, "column_mapping": `{"symbol":"symbol","side":"side","quantity":"quantity","price":"price","executed_at":"executed_at"}`}
	commit := func(want int) {
		rec := httptest.NewRecorder()
		s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "fills.csv", csv, fields))
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
		var res struct {
			Inserted int   `json:"inserted"`
			Errors   []any `json:"errors"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
		require.Equal(t, want, res.Inserted)
		require.Empty(t, res.Errors)
	}
	commit(2)
	rec := do(s, http.MethodGet, "/api/v1/executions?account_id="+acc, "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var fills []struct {
		ID string `json:"id"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &fills))
	require.Len(t, fills, 2)
	for _, fill := range fills {
		rec = do(s, http.MethodPatch, "/api/v1/executions/"+fill.ID, `{"side":"buy","quantity":1,"price":10,"fees":0,"executed_at":"2026-01-01T10:00:00Z"}`, tok)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}
	commit(0)
}

func TestImportLegacyRepeatedFillReportsError(t *testing.T) {
	s := testServer(t)
	tok := registerAndLogin(t, s, "legacy-occ@x.com")
	acc := accountID(t, s, tok)
	rec := do(s, http.MethodPost, "/api/v1/executions", `{"account_id":"`+acc+`","symbol":"AAPL","instrument_type":"stock","side":"buy","quantity":1,"price":10,"executed_at":"2026-01-01T10:00:00Z"}`, tok)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	csv := "symbol,side,quantity,price,executed_at\nAAPL,buy,1,10,2026-01-01T10:00:00Z\nAAPL,buy,1,10,2026-01-01T10:00:00Z\n"
	rec = httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, multipartFileReq(t, "/api/v1/imports/commit", tok, "fills.csv", csv, map[string]string{"account_id": acc, "column_mapping": `{"symbol":"symbol","side":"side","quantity":"quantity","price":"price","executed_at":"executed_at"}`}))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var res struct {
		Inserted int `json:"inserted"`
		Skipped  int `json:"skipped"`
		Errors   []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &res))
	require.Zero(t, res.Inserted)
	require.Equal(t, 2, res.Skipped)
	require.Len(t, res.Errors, 1)
	require.Contains(t, res.Errors[0].Message, "legacy import")
}
