package predictionvalidation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/marketdata"
)

func TestHTTPDataNeedsNoValidationAdapter(t *testing.T) {
	source := &evidenceProvider{name: "archive"}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "285A.T", r.URL.Query().Get("symbol"))
		from, err := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
		require.NoError(t, err)
		to, err := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
		require.NoError(t, err)
		response, err := source.FetchResponse(r.Context(), marketdata.Request{Symbol: "285A", Interval: r.URL.Query().Get("interval"), From: from, To: to})
		require.NoError(t, err)
		bars := []map[string]any{}
		for _, b := range response.Bars {
			bars = append(bars, map[string]any{"timestamp": time.Unix(b.Time, 0).UTC(), "market_date": b.MarketDate, "open": b.Open, "high": b.High, "low": b.Low, "close": b.Close, "volume": b.Volume, "fetched_at": response.FetchedAt})
		}
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"symbol": "285A.T", "interval": "D", "timezone": response.Timezone, "source": response.Source, "adjustment_status": response.AdjustmentStatus, "bars": bars}))
	}))
	defer server.Close()
	e := Engine{Market: marketdata.NewService(nil, marketdata.NewHTTPProvider(server.URL, ""))}
	result := e.Evaluate(context.Background(), input(), 3, instant("2026-12-01T00:00:00Z"))
	require.Equal(t, "validated", result.Status, result.Reason)
	require.Equal(t, "http", result.Evidence.Provider)
	require.Equal(t, "archive", result.Evidence.Source)
	require.NotNil(t, result.Evidence.Bars[0].FetchedAt)
}

func TestHTTPUnsupportedResolution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		_, _ = w.Write([]byte(`{"error":{"code":"unsupported_interval"}}`))
	}))
	defer server.Close()
	e := Engine{Market: marketdata.NewService(nil, marketdata.NewHTTPProvider(server.URL, ""))}
	r := e.Evaluate(context.Background(), input(), 1, instant("2026-12-01T00:00:00Z"))
	require.Equal(t, "unsupported_resolution", r.Reason)
}
