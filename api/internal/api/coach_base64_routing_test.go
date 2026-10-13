package api_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/ocr"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

const foldedOwnerKey = "q7M2n9R4t6V8x1Z3"
const foldedOwnerPrompt = "Use the private risk limits from owner instructions 295."

func TestSecondIndependentFoldedBase64Routes(t *testing.T) {
	testFoldedCoachRoutes(t, foldedOwnerKey, false)
}
func TestSecondIndependentFoldedPromptRoutes(t *testing.T) {
	testFoldedCoachRoutes(t, foldedOwnerPrompt, false)
}
func TestCoachFoldedAuthorizationRoutes(t *testing.T) {
	testFoldedCoachRoutes(t, "Authorization: Bearer "+foldedOwnerKey, false)
}
func TestCoachLegitimateBase64Routes(t *testing.T) {
	testFoldedCoachRoutes(t, "Keep risk controlled.", true)
}

func testFoldedCoachRoutes(t *testing.T, secret string, safe bool) {
	const key = "q7M2n9R4t6V8x1Z3"
	const prompt = "Use the private risk limits from owner instructions 295."
	f := newIsolationFixture(t)
	owner := f.seed(t, "owner", "OWN295", 2)
	member := f.seed(t, "member", "MEM295", 3)
	q := store.NewForDriver(f.conn, "sqlite")
	for _, variant := range []struct {
		name, separator string
		plain           bool
	}{
		{"LF JSON", "\n", false}, {"CRLF JSON", "\r\n", false}, {"LF plain", "\n", true}, {"CRLF plain", "\r\n", true},
	} {
		text := `{"value":"` + secret + `-tail"}`
		if variant.plain {
			text = secret
		}
		encoded := base64.StdEncoding.EncodeToString([]byte(text))
		var folded strings.Builder
		for i := 0; i < len(encoded); i += 16 {
			if i > 0 {
				folded.WriteString(variant.separator)
			}
			folded.WriteString(encoded[i:min(i+16, len(encoded))])
		}
		decoded, err := base64.StdEncoding.DecodeString(folded.String())
		require.NoError(t, err)
		require.True(t, strings.Contains(string(decoded), secret), "standard decoder must recover the configured text")
		for _, mode := range []string{"stream-failure", "stream-success", "json-success", "fallback-success"} {
			for _, identity := range []string{"jwt", "pat"} {
				t.Run(variant.name+"/"+mode+"/"+identity, func(t *testing.T) {
					var calls atomic.Int64
					var logs strings.Builder
					payload := isolationJSON(t, map[string]any{"notes": []any{map[string]string{"tone": "tip", "headline": "Provider result", "detail": folded.String()}}, "next_action": "Record the trade."})
					remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Header.Get("Authorization") != "Bearer "+key {
							t.Error("owner transport credential not received")
						}
						var request struct {
							Stream   bool
							Messages []struct{ Content string }
						}
						if json.NewDecoder(r.Body).Decode(&request) != nil {
							t.Error("provider request malformed")
							return
						}
						found := false
						for _, m := range request.Messages {
							found = found || strings.Contains(m.Content, prompt)
							if strings.Contains(m.Content, key) {
								t.Error("transport key in model message")
							}
						}
						if !found {
							t.Error("required private instructions not received")
						}
						w.Header().Set("X-Provider-Secret", key)
						if mode == "fallback-success" && request.Stream {
							w.WriteHeader(400)
							return
						}
						if !request.Stream {
							fmt.Fprint(w, isolationJSON(t, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": payload}}}}))
							return
						}
						w.Header().Set("Content-Type", "text/event-stream")
						text := payload
						if mode == "stream-failure" {
							text = strings.TrimSuffix(payload, "}") + ","
						}
						for i := 0; i < len(text); i += 7 {
							fmt.Fprintf(w, "data: %s\n\n", isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": text[i:min(i+7, len(text))]}}}}))
							w.(http.Flusher).Flush()
						}
						if mode == "stream-failure" {
							fmt.Fprint(w, `data: {"error":{"message":"aborted"}}`+"\n\n")
						} else {
							fmt.Fprint(w, "data: [DONE]\n\n")
						}
					}))
					defer remote.Close()
					isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": remote.URL, "api_key": key, "custom_prompt": prompt, "model": "fixture"}), owner.session, 200)
					s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(&logs, nil)), CoachDefaults: ocr.VisionConfig{HTTPClient: remote.Client()}})
					token := member.session
					if identity == "pat" {
						token = member.pat
					}
					require.Equal(t, 403, do(s, "GET", "/api/v1/settings/coach", "", token).Code)
					before := f.databaseState(t)
					suffix := "/coach/stream"
					if mode == "json-success" {
						suffix = "/coach"
					}
					rec := do(s, "POST", "/api/v1/trades/"+member.trade+suffix, `{}`, token)
					require.Equal(t, 200, rec.Code)
					if mode == "fallback-success" {
						require.EqualValues(t, 3, calls.Load())
					} else {
						require.EqualValues(t, 1, calls.Load())
					}
					if strings.Contains(fmt.Sprint(rec.Header())+logs.String(), key) {
						t.Error("credential in headers/logs")
					}
					if suffix == "/coach/stream" {
						frames := sseFrames(t, rec.Body.String())
						for _, frame := range frames {
							var value any
							require.NoError(t, json.Unmarshal([]byte(frame[1]), &value))
							if !safe && secondReviewContainsDecoded(value, secret) {
								t.Error("client-visible SSE event exposes line-folded owner key: " + frame[0])
							}
						}
						if mode == "stream-failure" {
							require.Equal(t, "error", frames[len(frames)-1][0])
							require.Contains(t, frames[len(frames)-1][1], "coach generation failed")
						}
					} else {
						var value any
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &value))
						if !safe && secondReviewContainsDecoded(value, secret) {
							t.Error("client-visible JSON exposes line-folded owner key")
						}
					}
					after := f.databaseState(t)
					if (!safe || mode == "stream-failure") && before["coach_reviews"] != after["coach_reviews"] {
						t.Error("unsafe review persisted")
					}
					if safe && mode != "stream-failure" {
						require.NotEqual(t, before["coach_reviews"], after["coach_reviews"], "safe review must persist")
					}
					delete(before, "coach_reviews")
					delete(after, "coach_reviews")
					require.Equal(t, before, after, "unrelated tables unchanged")
					_, err := q.CreateCoachReview(t.Context(), store.CreateCoachReviewParams{ID: "legacy-folded-295", UserID: member.id, TradeID: member.trade, Model: "fixture", Notes: isolationJSON(t, []map[string]string{{"headline": "Legacy", "detail": folded.String()}})})
					require.NoError(t, err)
					beforeRead := f.databaseState(t)
					history := do(s, "GET", "/api/v1/trades/"+member.trade+"/coach/reviews", "", token)
					require.Equal(t, 200, history.Code)
					var value any
					require.NoError(t, json.Unmarshal(history.Body.Bytes(), &value))
					if !safe && secondReviewContainsDecoded(value, secret) {
						t.Error("history readback exposes line-folded owner secret")
					}
					require.Equal(t, beforeRead, f.databaseState(t), "history hides rows without rewriting")
					historyRows := value.(map[string]any)["reviews"].([]any)
					if !safe {
						require.Empty(t, historyRows, "legacy unsafe row must be hidden")
					}
					if safe {
						require.NotEmpty(t, historyRows, "legitimate encoded history remains usable")
					}
					_, err = f.conn.Exec("DELETE FROM coach_reviews WHERE user_id=?", member.id)
					require.NoError(t, err)
				})
			}
		}
	}
}

func secondReviewContainsDecoded(value any, key string) bool {
	switch v := value.(type) {
	case string:
		if strings.Contains(v, key) {
			return true
		}
		decoded, err := base64.StdEncoding.DecodeString(v)
		return err == nil && strings.Contains(string(decoded), key)
	case map[string]any:
		for _, child := range v {
			if secondReviewContainsDecoded(child, key) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if secondReviewContainsDecoded(child, key) {
				return true
			}
		}
	}
	return false
}
