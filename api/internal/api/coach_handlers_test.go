package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestTradeCoach_offWhenDisabled(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-off@example.com")
	acc := accountID(t, s, tok)
	tradeID := closedTradeID(t, s, tok, acc)

	rec := do(s, http.MethodPost, "/api/v1/trades/"+tradeID+"/coach", "", tok)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	var got map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
	require.Equal(t, "off", got["source"])
	require.Equal(t, []any{}, got["notes"])
}

func TestTradeCoachReviews_scopedToOwner(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-owner@example.com")
	acc := accountID(t, s, tok)
	tradeID := closedTradeID(t, s, tok, acc)

	other := registerAndLogin(t, s, "coach-other@example.com")
	rec := do(s, http.MethodGet, "/api/v1/trades/"+tradeID+"/coach/reviews", "", other)
	require.Equal(t, http.StatusNotFound, rec.Code, rec.Body.String())
}

func TestTradeCoachReviews_notFound(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-rev-404@example.com")
	rec := do(s, http.MethodGet, "/api/v1/trades/does-not-exist/coach/reviews", "", tok)
	require.Equal(t, http.StatusNotFound, rec.Code)
}

// A failed review is not stored — history should hold advice, not errors.
func TestTradeCoach_notFound(t *testing.T) {
	s := testServerWithOCR(t, ocr.VisionConfig{})
	tok := registerAndLogin(t, s, "coach-404@example.com")
	rec := do(s, http.MethodPost, "/api/v1/trades/missing/coach", "", tok)
	require.Equal(t, http.StatusNotFound, rec.Code)
}
