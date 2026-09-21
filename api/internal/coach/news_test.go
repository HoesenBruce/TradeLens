package coach

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

const validNewsAnalysis = `{"summary":"Supply expansion","category":"industry","assets":[{"asset_type":"stock","symbol":"285A","market":"JP","exchange":"TSE","display_name":"Test","direction":"bullish","confidence":65,"reasoning":"Potential demand","catalysts":"Orders","risks":"Execution","horizons":[1,5]},{"asset_type":"etf","symbol":"SPY","market":"US","exchange":"NYSE","display_name":"S&P","direction":"neutral","confidence":0,"reasoning":"Limited exposure","catalysts":"","risks":"","horizons":[20]}]}`

func TestAnalyzeNews(t *testing.T) {
	for _, fallback := range []bool{false, true} {
		t.Run(map[bool]string{false: "schema", true: "fallback"}[fallback], func(t *testing.T) {
			var formats []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				require.Equal(t, "Bearer test", r.Header.Get("Authorization"))
				var req chatRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				formats = append(formats, req.ResponseFormat.Type)
				require.Equal(t, "news-model", req.Model)
				require.Contains(t, req.Messages[1].Content, "285A")
				require.NotContains(t, req.Messages[0].Content, "custom coach prompt")
				if fallback && len(formats) == 1 {
					w.WriteHeader(400)
					return
				}
				require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": validNewsAnalysis}}}}))
			}))
			defer srv.Close()
			got, err := AnalyzeNews(context.Background(), ocr.VisionConfig{Enabled: true, BaseURL: srv.URL, APIKey: "test", Model: "news-model", CustomPrompt: "custom coach prompt", HTTPClient: srv.Client()}, "285A news")
			require.NoError(t, err)
			require.Len(t, got.Assets, 2)
			require.Equal(t, "285A", got.Assets[0].Symbol)
			if fallback {
				require.Equal(t, []string{"json_schema", "json_object"}, formats)
			} else {
				require.Equal(t, []string{"json_schema"}, formats)
			}
		})
	}
}

func TestNewsAnalysisRejectsEntireMalformedResult(t *testing.T) {
	cases := []string{`{}`, `null`, validNewsAnalysis + `{}`, strings.Replace(validNewsAnalysis, `"confidence":65`, `"confidence":null`, 1), strings.Replace(validNewsAnalysis, `"risks":"Execution",`, "", 1), strings.Replace(validNewsAnalysis, `"market":"JP"`, `"market":null`, 1), strings.Replace(validNewsAnalysis, `"confidence":65`, `"confidence":101`, 1), strings.Replace(validNewsAnalysis, `[1,5]`, `[2]`, 1), strings.Replace(validNewsAnalysis, `[1,5]`, `[1,1]`, 1), strings.Replace(validNewsAnalysis, `"neutral"`, `"unknown"`, 1), strings.Replace(validNewsAnalysis, `"summary":`, `"extra":true,"summary":`, 1)}
	for _, raw := range cases {
		got, err := parseNewsAnalysis(raw)
		require.Error(t, err, raw)
		require.Empty(t, got.Assets)
	}
}

func TestNewsAnalysisTimeoutAndAuth(t *testing.T) {
	for _, status := range []int{401, 429, 500, 200} {
		calls := 0
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			if status == 200 {
				time.Sleep(80 * time.Millisecond)
				return
			}
			w.WriteHeader(status)
		}))
		_, err := AnalyzeNews(context.Background(), ocr.VisionConfig{Enabled: true, BaseURL: srv.URL, APIKey: "test", Timeout: 30 * time.Millisecond, HTTPClient: srv.Client()}, "news")
		require.Error(t, err)
		if status == 200 {
			require.ErrorIs(t, err, ErrTimeout)
		}
		srv.Close()
		require.Equal(t, 1, calls)
	}
	_, err := AnalyzeNews(context.Background(), ocr.VisionConfig{}, "news")
	require.ErrorIs(t, err, ErrUnavailable)
}
