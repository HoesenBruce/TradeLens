package api_test

import (
	"encoding/json"
	"fmt"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/db"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestManualPredictions(t *testing.T) {
	conn, err := db.Open(filepath.Join(t.TempDir(), "prediction.db"))
	require.NoError(t, err)
	defer conn.Close()
	require.NoError(t, db.Migrate(conn))
	q := store.NewForDriver(conn, db.DriverSQLite)
	jwt := auth.NewJWT("prediction-test-secret")
	s := api.New(api.Deps{JWT: jwt, Auth: auth.NewService(q, jwt, true), Store: q, Trades: trades.NewService(q)})
	token := registerAndLogin(t, s, "prediction@example.com")
	created := do(s, http.MethodPost, "/api/v1/news", `{"title":"Catalyst","source":"manual","published_at":"2026-09-20T00:00:00Z","assets":[{"asset_type":"stock","symbol":"285A"}]}`, token)
	require.Equal(t, 201, created.Code, created.Body.String())
	var n newsResponse
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &n))
	path := "/api/v1/news/" + n.ID + "/predictions"
	body := fmt.Sprintf(`{"news_asset_id":%q,"direction":"bullish","confidence":80,"reasoning":"Demand","catalysts":"Launch","risks":"Delay","invalidation":"Cancelled","horizons":[1,5,20]}`, n.Assets[0].ID)
	result := do(s, http.MethodPost, path, body, token)
	require.Equal(t, 201, result.Code, result.Body.String())
	var p struct {
		ID         string  `json:"id"`
		Source     string  `json:"source"`
		Confidence *int64  `json:"confidence"`
		Horizons   []int64 `json:"horizons"`
		Direction  string  `json:"direction"`
	}
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &p))
	require.Equal(t, "user", p.Source)
	require.Equal(t, []int64{1, 5, 20}, p.Horizons)
	id := p.ID
	_, err = conn.Exec("INSERT INTO predictions (id, news_asset_id, source, direction, reasoning) VALUES ('ai-prediction', ?, 'ai', 'neutral', 'AI original')", n.Assets[0].ID)
	require.NoError(t, err)
	require.Equal(t, 404, do(s, http.MethodPatch, path+"/ai-prediction", body, token).Code)
	require.Equal(t, 404, do(s, http.MethodDelete, path+"/ai-prediction", "", token).Code)
	var reasoning string
	require.NoError(t, conn.QueryRow("SELECT reasoning FROM predictions WHERE id = 'ai-prediction'").Scan(&reasoning))
	require.Equal(t, "AI original", reasoning)
	_, err = conn.Exec("DELETE FROM predictions WHERE id = 'ai-prediction'")
	require.NoError(t, err)

	for _, bad := range []string{`"confidence":101`, `"confidence":-1`, `"confidence":1.5`, `"source":"ai"`, `"direction":"up"`, `"horizons":[]`, `"horizons":[2]`, `"horizons":[1,1]`} {
		var input map[string]any
		require.NoError(t, json.Unmarshal([]byte(body), &input))
		var patch map[string]any
		require.NoError(t, json.Unmarshal([]byte("{"+bad+"}"), &patch))
		for k, v := range patch {
			input[k] = v
		}
		b, _ := json.Marshal(input)
		require.Equal(t, 400, do(s, http.MethodPatch, path+"/"+id, string(b), token).Code, bad)
	}
	other := registerAndLogin(t, s, "prediction-other@example.com")
	require.Equal(t, 404, do(s, http.MethodPost, path, body, other).Code)
	require.Equal(t, 404, do(s, http.MethodPatch, path+"/"+id, body, other).Code)
	require.Equal(t, 404, do(s, http.MethodDelete, path+"/"+id, "", other).Code)
	require.Equal(t, 404, do(s, http.MethodPatch, "/api/v1/news/missing/predictions/"+id, body, token).Code)
	result = do(s, http.MethodPatch, path+"/"+id, `{"direction":"bearish","confidence":null,"reasoning":"Revised","horizons":[3,10]}`, token)
	require.Equal(t, 200, result.Code, result.Body.String())
	require.NoError(t, json.Unmarshal(result.Body.Bytes(), &p))
	require.Nil(t, p.Confidence)
	require.Equal(t, []int64{3, 10}, p.Horizons)
	read := do(s, http.MethodGet, "/api/v1/news/"+n.ID, "", token)
	require.Contains(t, read.Body.String(), `"direction":"bearish"`)
	require.Contains(t, read.Body.String(), `"horizons":[3,10]`)
	require.Equal(t, 204, do(s, http.MethodDelete, path+"/"+id, "", token).Code)
	require.Equal(t, 404, do(s, http.MethodDelete, path+"/"+id, "", token).Code)
	read = do(s, http.MethodGet, "/api/v1/news/"+n.ID, "", token)
	require.Contains(t, read.Body.String(), `"predictions":[]`)
}
