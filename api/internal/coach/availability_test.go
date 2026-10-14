package coach

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func TestLegacyEntryPointsFailClosed(t *testing.T) {
	var calls atomic.Int64
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(400) // would trigger schema/stream fallback in the legacy engine
	}))
	defer provider.Close()
	cfg := ocr.VisionConfig{Enabled: true, BaseURL: provider.URL, APIKey: "synthetic-key", Model: "synthetic-model", CustomPrompt: "synthetic-private-prompt", HTTPClient: provider.Client()}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, canceled := range []bool{false, true} {
		if canceled {
			cancel()
		}
		_, err := GenerateReview(ctx, cfg, TradeContext{})
		require.ErrorIs(t, err, ErrUnavailable)
		_, err = GenerateReviewStream(ctx, cfg, TradeContext{}, func(Note) { t.Error("unsafe callback invoked") })
		require.ErrorIs(t, err, ErrUnavailable)
		_, err = AnalyzeNews(ctx, cfg, "synthetic news")
		require.ErrorIs(t, err, ErrUnavailable)
		_, err = requestContent(ctx, cfg, nil, objectFormat())
		require.ErrorIs(t, err, ErrUnavailable)
		_, err = requestReview(ctx, cfg, nil, schemaFormat())
		require.ErrorIs(t, err, ErrUnavailable)
		_, err = streamReview(ctx, cfg, nil, objectFormat(), func(Note) { t.Error("unsafe callback invoked") })
		require.ErrorIs(t, err, ErrUnavailable)
	}
	require.Zero(t, calls.Load())
}
