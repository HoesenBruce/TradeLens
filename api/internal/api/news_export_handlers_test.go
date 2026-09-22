package api_test

import (
	"encoding/json"
	"fmt"
	"strings"
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

func TestBatchNewsExport(t *testing.T) {
	s := testServer(t)
	token := registerAndLogin(t, s, "export-batch@example.com")
	path := "/api/v1/news/export"
	empty := do(s, "GET", path, "", token)
	require.Equal(t, 200, empty.Code, empty.Body.String())
	require.Contains(t, empty.Body.String(), "count: 0")
	require.Contains(t, empty.Body.String(), "No News Theses match")
	var ids []string
	for _, tc := range []struct{ title, symbol, date string }{{"Older thesis", "285A", "2026-09-19"}, {"Newer thesis", "AAPL", "2026-09-20"}} {
		created := do(s, "POST", "/api/v1/news", fmt.Sprintf(`{"title":%q,"source":"manual","published_at":"%sT00:00:00Z","category":"Industry","assets":[{"asset_type":"stock","symbol":%q}]}`, tc.title, tc.date, tc.symbol), token)
		require.Equal(t, 201, created.Code, created.Body.String())
		var n newsResponse
		require.NoError(t, json.Unmarshal(created.Body.Bytes(), &n))
		ids = append(ids, n.ID)
		pred := do(s, "POST", "/api/v1/news/"+n.ID+"/predictions", fmt.Sprintf(`{"news_asset_id":%q,"direction":"bullish","horizons":[1,5]}`, n.Assets[0].ID), token)
		require.Equal(t, 201, pred.Code, pred.Body.String())
	}
	all := do(s, "GET", path, "", token)
	require.Equal(t, 200, all.Code, all.Body.String())
	require.Equal(t, `attachment; filename="news-theses.md"`, all.Header().Get("Content-Disposition"))
	require.Contains(t, all.Body.String(), "count: 2")
	require.Less(t, strings.Index(all.Body.String(), "# Newer thesis"), strings.Index(all.Body.String(), "# Older thesis"))
	require.Equal(t, all.Body.String(), do(s, "GET", path, "", token).Body.String())
	for _, query := range []string{"symbol=285a", "from=2026-09-19&to=2026-09-19", "source=user&symbol=285A&asset_type=stock&category=Industry&horizon=5", "id=" + ids[0] + "&id=" + ids[0]} {
		out := do(s, "GET", path+"?"+query, "", token)
		require.Equal(t, 200, out.Code, out.Body.String())
		require.Contains(t, out.Body.String(), "count: 1")
		require.Contains(t, out.Body.String(), "# Older thesis")
		require.NotContains(t, out.Body.String(), "Newer thesis")
	}
	for _, query := range []string{"symbol=missing", "source=ai", "horizon=20", "id=" + ids[0] + "&symbol=AAPL"} {
		out := do(s, "GET", path+"?"+query, "", token)
		require.Equal(t, 200, out.Code, out.Body.String())
		require.Contains(t, out.Body.String(), "count: 0")
	}
	for _, query := range []string{"source=bad", "horizon=2", "horizon=0", "horizon=abc", "from=bad", "from=2026-09-20&to=2026-09-19", "asset_type=coin", "id="} {
		require.Equal(t, 400, do(s, "GET", path+"?"+query, "", token).Code, query)
	}
	require.Equal(t, 401, do(s, "GET", path, "", "").Code)
	other := registerAndLogin(t, s, "batch-other@example.com")
	require.Contains(t, do(s, "GET", path, "", other).Body.String(), "count: 0")
	require.Equal(t, 404, do(s, "GET", path+"?id="+ids[0], "", other).Code)
	require.Equal(t, 404, do(s, "GET", path+"?id=missing", "", token).Code)
}
