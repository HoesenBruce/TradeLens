package api_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCrossUserIsolationCoachErrorsHideOwnerConfiguration(t *testing.T) {
	f := newIsolationFixture(t)
	owner := registerAndLogin(t, f.s, "owner@recovery.invalid")
	member := registerAndLogin(t, f.s, "member@recovery.invalid")
	account := accountID(t, f.s, member)
	trade := closedTradeID(t, f.s, member, account)
	const key = "synthetic-owner-coach-key"
	const prompt = "synthetic-owner-private-prompt"
	var received atomic.Int64
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer "+key || !strings.Contains(string(body), prompt) {
			t.Error("mock did not receive the configured owner credential and prompt")
		}
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{
			"message": "API key=" + key + "; Authorization=" + r.Header.Get("Authorization") + "; config=" + prompt,
		}})
	}))
	t.Cleanup(remote.Close)
	isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{
		"enabled": true, "base_url": remote.URL, "api_key": key, "model": "synthetic", "custom_prompt": prompt,
	}), owner, 200)
	pat := isolationObject(t, f.s, "POST", "/access-tokens", `{"name":"member"}`, member, 201)["token"].(string)
	for name, token := range map[string]string{"jwt": member, "pat": pat} {
		for _, suffix := range []string{"/coach", "/coach/stream"} {
			t.Run(name+suffix, func(t *testing.T) {
				require.Equal(t, 403, do(f.s, "GET", "/api/v1/settings/coach", "", token).Code)
				before := f.databaseState(t)
				response := do(f.s, "POST", "/api/v1/trades/"+trade+suffix, `{}`, token)
				require.Equal(t, 200, response.Code)
				require.Equal(t, before, f.databaseState(t), "failed generation must not persist a review")
				for _, secret := range []string{key, "Authorization=Bearer", prompt} {
					if strings.Contains(response.Body.String(), secret) {
						t.Error("member response disclosed a synthetic owner configuration secret")
					}
				}
			})
		}
	}
	require.EqualValues(t, 4, received.Load())
}
