package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"testing"
)

func TestNewsPerformanceAPI(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "performance-api@example.com")
	rec := do(s, "GET", "/api/v1/news/performance", "", token)
	require.Equal(t, 200, rec.Code, rec.Body.String())
	var out predictionvalidation.Performance
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, "prediction_horizon", out.Unit)
	require.Zero(t, out.Counts.Total)
	require.NotNil(t, out.BySource)
	for _, query := range []string{"source=bad", "horizon=2", "horizon=abc", "horizon=0", "asset_type=coin", "from=bad", "from=2026-09-02&to=2026-09-01"} {
		require.Equal(t, 400, do(s, "GET", "/api/v1/news/performance?"+query, "", token).Code, query)
	}
	require.Equal(t, 401, do(s, "GET", "/api/v1/news/performance", "", "").Code)
	require.Equal(t, 200, do(s, "GET", "/api/v1/news/performance?source=user&symbol=285A&category=Industry&asset_type=stock&horizon=3&from=2026-09-01&to=2026-09-21", "", token).Code)
}
