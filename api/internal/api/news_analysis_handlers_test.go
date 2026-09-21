package api_test

import (
	"encoding/json"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewsAnalysisReview(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "news-ai@example.com")
	created := do(s, "POST", "/api/v1/news", `{"title":"285A supply","source":"manual","published_at":"2026-09-21T00:00:00Z","summary":"Manual summary","category":"Manual","assets":[{"asset_type":"stock","symbol":"285A","market":"JP"}]}`, token)
	require.Equal(t, 201, created.Code)
	var news newsResponse
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &news))
	path := "/api/v1/news/" + news.ID
	before := do(s, "GET", path, "", token).Body.String()
	require.Equal(t, 503, do(s, "POST", path+"/analyze", "", token).Code)
	content := `{"summary":"AI summary","category":"Industry","assets":[{"asset_type":"stock","symbol":"285A","market":"JP","exchange":"TSE","display_name":"Test","direction":"bullish","confidence":70,"reasoning":"AI reason","catalysts":"","risks":"","horizons":[1,5]}]}`
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewEncoder(w).Encode(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": content}}}}))
	}))
	defer provider.Close()
	settings := `{"enabled":true,"base_url":"` + provider.URL + `","model":"test","api_key":"fixture"}`
	require.Equal(t, 200, do(s, "PUT", "/api/v1/settings/coach", settings, token).Code)
	analyzed := do(s, "POST", path+"/analyze", "", token)
	require.Equal(t, 200, analyzed.Code, analyzed.Body.String())
	require.JSONEq(t, before, do(s, "GET", path, "", token).Body.String())
	content = `{"summary":"partial"}`
	require.Equal(t, 502, do(s, "POST", path+"/analyze", "", token).Code)
	require.JSONEq(t, before, do(s, "GET", path, "", token).Body.String())
	other := registerAndLogin(t, s, "news-ai-other@example.com")
	require.Equal(t, 404, do(s, "POST", path+"/analyze", "", other).Code)
	review := `{"summary":"Edited AI summary","expected_summary":"Manual summary","assets":[{"source":"ai","include_prediction":true,"asset_type":"stock","symbol":"285A","market":"JP","direction":"bearish","confidence":50,"reasoning":"Edited AI reason","horizons":[3]},{"source":"user","include_prediction":false,"asset_type":"etf","symbol":"1306"}]}`
	require.Equal(t, 404, do(s, "POST", path+"/analysis/accept", review, other).Code)
	invalid := `{"summary":"must roll back","expected_summary":"Manual summary","assets":[{"source":"ai","asset_type":"stock","symbol":"SPY"},{"source":"ai","include_prediction":true,"asset_type":"stock","symbol":"285A","direction":"bullish","confidence":101,"reasoning":"x","horizons":[1]}]}`
	require.Equal(t, 400, do(s, "POST", path+"/analysis/accept", invalid, token).Code)
	require.JSONEq(t, before, do(s, "GET", path, "", token).Body.String())
	saved := do(s, "POST", path+"/analysis/accept", review, token)
	require.Equal(t, 200, saved.Code, saved.Body.String())
	var out struct {
		Summary     string
		Category    string
		Assets      []struct{ ID, Source string }
		Predictions []struct {
			Source, Reasoning string
			Horizons          []int64
		}
	}
	require.NoError(t, json.Unmarshal(saved.Body.Bytes(), &out))
	require.Equal(t, "Edited AI summary", out.Summary)
	require.Equal(t, "Manual", out.Category)
	require.Len(t, out.Assets, 3)
	require.Len(t, out.Predictions, 1)
	require.Equal(t, "ai", out.Predictions[0].Source)
	require.Equal(t, []int64{3}, out.Predictions[0].Horizons)
	require.Equal(t, 409, do(s, "POST", path+"/analysis/accept", review, token).Code)
	require.JSONEq(t, saved.Body.String(), do(s, "GET", path, "", token).Body.String())
}
