package api_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSingleNewsExport(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "export-news@example.com")
	created := do(s, "POST", "/api/v1/news", `{"title":"Export thesis","source":"manual","published_at":"2026-09-20T00:00:00Z","notes":"Review notes","assets":[{"asset_type":"stock","symbol":"285A"}]}`, token)
	require.Equal(t, 201, created.Code, created.Body.String())
	var n newsResponse
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &n))
	pred := do(s, "POST", "/api/v1/news/"+n.ID+"/predictions", fmt.Sprintf(`{"news_asset_id":%q,"direction":"bullish","reasoning":"Demand","horizons":[1,5]}`, n.Assets[0].ID), token)
	require.Equal(t, 201, pred.Code, pred.Body.String())
	path := "/api/v1/news/" + n.ID + "/export"
	out := do(s, "GET", path, "", token)
	require.Equal(t, 200, out.Code, out.Body.String())
	require.Equal(t, "text/markdown; charset=utf-8", out.Header().Get("Content-Type"))
	require.Equal(t, fmt.Sprintf(`attachment; filename="news-thesis-%s.md"`, n.ID), out.Header().Get("Content-Disposition"))
	require.Contains(t, out.Body.String(), "# Export thesis")
	require.Contains(t, out.Body.String(), "User prediction")
	require.Contains(t, out.Body.String(), "5 trading days: pending")
	require.Contains(t, out.Body.String(), "Review notes")
	require.Equal(t, out.Body.String(), do(s, "GET", path, "", token).Body.String())
	require.Equal(t, 401, do(s, "GET", path, "", "").Code)
	other := registerAndLogin(t, s, "export-other@example.com")
	require.Equal(t, 404, do(s, "GET", path, "", other).Code)
	require.Equal(t, 404, do(s, "GET", "/api/v1/news/missing/export", "", token).Code)
}
