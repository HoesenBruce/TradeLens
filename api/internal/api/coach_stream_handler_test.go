package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestTradeCoachStream_offWhenDisabled(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-stream-off@example.com")
	acc := accountID(t, s, tok)
	tradeID := closedTradeID(t, s, tok, acc)

	rec := do(s, http.MethodPost, "/api/v1/trades/"+tradeID+"/coach/stream", "", tok)
	require.Equal(t, http.StatusOK, rec.Code)
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "off", got["source"])
}

func TestTradeCoachStream_notFound(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-stream-404@example.com")
	rec := do(s, http.MethodPost, "/api/v1/trades/nope/coach/stream", "", tok)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
