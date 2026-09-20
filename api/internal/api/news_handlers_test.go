package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
)

type newsAssetResponse struct {
	ID          string `json:"id"`
	NewsID      string `json:"news_id"`
	AssetType   string `json:"asset_type"`
	Symbol      string `json:"symbol"`
	Exchange    string `json:"exchange"`
	DisplayName string `json:"display_name"`
}

type newsResponse struct {
	ID      string              `json:"id"`
	Title   string              `json:"title"`
	Summary string              `json:"summary"`
	Tags    []string            `json:"tags"`
	Assets  []newsAssetResponse `json:"assets"`
}

func TestNewsCRUDAndAssets(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "news-api@example.com")

	created := do(s, http.MethodPost, "/api/v1/news", `{
      "title":"Japan ETF launch","source":"manual","url":"https://example.com/news",
      "published_at":"2026-09-20T09:30:00Z","original_text":"original","notes":"watch",
      "tags":["ETF"," ETF "],
      "assets":[
        {"asset_type":"stock","symbol":"285a","market":"JP"},
        {"asset_type":"index","symbol":"N225","source":"ai"}
      ]
    }`, token)
	require.Equal(t, http.StatusCreated, created.Code, created.Body.String())
	var item newsResponse
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &item))
	require.NotEmpty(t, item.ID)
	require.Equal(t, []string{"ETF"}, item.Tags)
	require.Len(t, item.Assets, 2)
	require.ElementsMatch(t, []string{"285A", "N225"}, []string{item.Assets[0].Symbol, item.Assets[1].Symbol})
	require.NotEmpty(t, item.Assets[0].ID)

	listed := do(s, http.MethodGet, "/api/v1/news", "", token)
	require.Equal(t, http.StatusOK, listed.Code, listed.Body.String())
	var items []newsResponse
	require.NoError(t, json.Unmarshal(listed.Body.Bytes(), &items))
	require.Len(t, items, 1)
	require.Equal(t, item.ID, items[0].ID)
	require.Len(t, items[0].Assets, 2)

	updated := do(s, http.MethodPatch, "/api/v1/news/"+item.ID, `{
      "title":"Japan ETF launched","source":"manual","published_at":"2026-09-20T09:30:00Z",
      "summary":"new listing","category":"markets","tags":["ETF","Japan"]
    }`, token)
	require.Equal(t, http.StatusOK, updated.Code, updated.Body.String())
	require.NoError(t, json.Unmarshal(updated.Body.Bytes(), &item))
	require.Equal(t, "Japan ETF launched", item.Title)
	require.Equal(t, "new listing", item.Summary)
	require.Len(t, item.Assets, 2)

	added := do(s, http.MethodPost, "/api/v1/news/"+item.ID+"/assets", `{
      "asset_type":"ETF","symbol":"1306","display_name":"TOPIX ETF"
    }`, token)
	require.Equal(t, http.StatusCreated, added.Code, added.Body.String())
	var asset newsAssetResponse
	require.NoError(t, json.Unmarshal(added.Body.Bytes(), &asset))
	require.Equal(t, "etf", asset.AssetType)
	require.Equal(t, item.ID, asset.NewsID)

	changed := do(s, http.MethodPatch, "/api/v1/news/"+item.ID+"/assets/"+asset.ID, `{
      "asset_type":"etf","symbol":"1306","exchange":"TSE","display_name":"TOPIX ETF"
    }`, token)
	require.Equal(t, http.StatusOK, changed.Code, changed.Body.String())
	require.NoError(t, json.Unmarshal(changed.Body.Bytes(), &asset))
	require.Equal(t, "TSE", asset.Exchange)

	require.Equal(t, http.StatusNoContent, do(
		s, http.MethodDelete, "/api/v1/news/"+item.ID+"/assets/"+asset.ID, "", token,
	).Code)

	otherToken := registerAndLogin(t, s, "news-api-other@example.com")
	require.Equal(t, http.StatusNotFound, do(s, http.MethodGet, "/api/v1/news/"+item.ID, "", otherToken).Code)
	require.Equal(t, http.StatusNotFound, do(
		s, http.MethodPatch, "/api/v1/news/"+item.ID+"/assets/"+item.Assets[0].ID,
		`{"asset_type":"stock","symbol":"HACK"}`, otherToken,
	).Code)

	require.Equal(t, http.StatusNoContent, do(s, http.MethodDelete, "/api/v1/news/"+item.ID, "", token).Code)
	require.Equal(t, http.StatusNotFound, do(s, http.MethodGet, "/api/v1/news/"+item.ID, "", token).Code)
}

func TestNewsValidationAndMissingRecords(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "news-errors@example.com")

	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"title", `{"source":"manual","published_at":"2026-09-20T09:30:00Z"}`, "title is required"},
		{"url", `{"title":"x","source":"manual","published_at":"2026-09-20T09:30:00Z","url":"ftp://example.com"}`, "url must be an HTTP(S) URL"},
		{"asset", `{"title":"x","source":"manual","published_at":"2026-09-20T09:30:00Z","assets":[{"asset_type":"coin","symbol":"BTC"}]}`, "asset_type must be stock, etf, or index"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rec := do(s, http.MethodPost, "/api/v1/news", tc.body, token)
			require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			var out struct {
				Error api.APIError `json:"error"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
			require.Equal(t, tc.want, out.Error.Message)
		})
	}

	const missing = "00000000-0000-0000-0000-000000000000"
	require.Equal(t, http.StatusNotFound, do(s, http.MethodGet, "/api/v1/news/"+missing, "", token).Code)
	require.Equal(t, http.StatusNotFound, do(s, http.MethodDelete, "/api/v1/news/"+missing, "", token).Code)
	require.Equal(t, http.StatusNotFound, do(
		s, http.MethodPost, "/api/v1/news/"+missing+"/assets",
		`{"asset_type":"stock","symbol":"285A"}`, token,
	).Code)
}
