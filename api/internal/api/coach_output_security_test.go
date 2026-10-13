package api_test

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/coach"
	"github.com/tradermemos/api/internal/ocr"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestIndependentCoachPartialFailureMustNotDisclose(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "owner", "ISOA", 2)
	b := f.seed(t, "member", "ISOB", 3)
	const key = "synthetic-owner-partial-failure-key-295"
	const prompt = "synthetic-owner-private-prompt-partial-295"
	var calls atomic.Int64
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+key {
			t.Error("owner credential not exercised")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), prompt) {
			t.Error("owner prompt not exercised")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		partial := isolationJSON(t, map[string]any{"notes": []any{map[string]string{"tone": "watch", "headline": "Provider rejection", "detail": "Invalid API key: " + key + "; Authorization: Bearer " + key + "; " + prompt}}})
		// A complete note arrives before the review is complete; generation then fails.
		partial = strings.TrimSuffix(partial, "}") + ","
		chunk := isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": partial}}}})
		fmt.Fprintf(w, "data: %s\n\n", chunk)
		w.(http.Flusher).Flush()
		fmt.Fprintf(w, "data: %s\n\n", isolationJSON(t, map[string]any{"error": map[string]string{"message": "generation aborted: " + key}}))
	}))
	defer remote.Close()
	q := store.NewForDriver(f.conn, "sqlite")
	isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": remote.URL, "api_key": key, "custom_prompt": prompt}), owner.session, 200)
	cfg := ocr.VisionConfig{HTTPClient: remote.Client()}
	s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), CoachDefaults: cfg})
	for name, token := range map[string]string{"jwt": b.session, "pat": b.pat} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, 403, do(s, "GET", "/api/v1/settings/coach", "", token).Code)
			before := f.databaseState(t)
			rec := do(s, "POST", "/api/v1/trades/"+b.trade+"/coach/stream", `{}`, token)
			require.Equal(t, 200, rec.Code)
			frames := sseFrames(t, rec.Body.String())
			// The unsafe preview is now rejected before any note is flushed.
			require.Len(t, frames, 1)
			require.Equal(t, "error", frames[0][0])
			require.Contains(t, frames[0][1], "coach generation failed")
			require.Equal(t, before, f.databaseState(t), "failed review was not persisted")
			for _, sentinel := range []struct{ name, value string }{{"owner API key", key}, {"owner private prompt", prompt}, {"Authorization header text", "Authorization: Bearer"}} {
				if strings.Contains(rec.Body.String()+fmt.Sprint(rec.Header()), sentinel.value) {
					t.Error("failed stream disclosed " + sentinel.name)
				}
			}
		})
	}
	require.EqualValues(t, 2, calls.Load())
}

func TestCoachOutputSecurityRouting(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "owner", "ISOA", 2)
	member := f.seed(t, "member", "ISOB", 3)
	const key = "synthetic+owner-private-key/295"
	const prompt = "Owner private prompt 295.\nDistinctive hidden instruction 295."
	type scenario struct {
		name, text                                string
		fail, safe, mixed, malformed, empty, next bool
	}
	var scenarios = []scenario{
		{name: "secret then failure", text: key, fail: true},
		{name: "secret then success", text: key},
		{name: "private prompt", text: prompt},
		{name: "prompt line", text: "Distinctive hidden instruction 295."},
		{name: "url encoded", text: url.QueryEscape(url.QueryEscape(key))},
		{name: "base64 envelope", text: base64.StdEncoding.EncodeToString([]byte(`{"failure":"` + key + `"}`))},
		{name: "hex", text: hex.EncodeToString([]byte(key))},
		{name: "multiline", text: strings.Join(strings.Split(key, ""), "\n")},
		{name: "nested diagnostic", text: `{"error":{"Authorization":"Bearer ` + key + `"}}`},
		{name: "unknown bearer", text: "Bearer unrelated-secret-token-295"},
		{name: "mixed", text: key, mixed: true},
		{name: "next action", text: key, next: true},
		{name: "split fields", text: key},
		{name: "oversize", text: strings.Repeat("x", (64<<10)+1)},
		{name: "safe failure", safe: true, fail: true},
		{name: "safe success", safe: true},
		{name: "empty chunks", safe: true, empty: true},
		{name: "malformed chunk", safe: true, malformed: true},
	}
	var unicodeText, htmlText strings.Builder
	for _, r := range key {
		fmt.Fprintf(&unicodeText, `\u%04x`, r)
		fmt.Fprintf(&htmlText, "&#%d;", r)
	}
	scenarios = append(scenarios, scenario{name: "unicode escaped", text: unicodeText.String()}, scenario{name: "HTML entities", text: htmlText.String()})
	q := store.NewForDriver(f.conn, "sqlite")
	for _, tc := range scenarios {
		for _, token := range []string{member.session, member.pat} {
			for _, suffix := range []string{"/coach/stream", "/coach"} {
				t.Run(tc.name+suffix, func(t *testing.T) {
					var calls atomic.Int64
					var logs strings.Builder
					note := map[string]string{"tone": "tip", "headline": "Safe first", "detail": "Keep risk controlled."}
					notes := []any{note, map[string]string{"tone": "tip", "headline": "Safe second", "detail": "Write a journal."}}
					if !tc.safe {
						unsafe := map[string]string{"tone": "warn", "headline": "Provider diagnostic", "detail": tc.text}
						if tc.name == "split fields" {
							unsafe["headline"] = key[:len(key)/2]
							unsafe["detail"] = key[len(key)/2:]
						}
						notes = []any{unsafe}
						if tc.mixed {
							notes = []any{note, unsafe}
						}
						if tc.next {
							notes = []any{note}
						}
					}
					next := "Record the next plan."
					if tc.next {
						next = tc.text
					}
					payload := isolationJSON(t, map[string]any{"notes": notes, "next_action": next})
					remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						calls.Add(1)
						if r.Header.Get("Authorization") != "Bearer "+key {
							t.Error("configured credential not used")
						}
						var request struct {
							Stream   bool
							Messages []struct{ Content string }
						}
						if json.NewDecoder(r.Body).Decode(&request) != nil {
							t.Error("request not decoded")
							return
						}
						foundPrompt := false
						for _, message := range request.Messages {
							if strings.Contains(message.Content, key) {
								t.Error("transport key entered model-visible messages")
							}
							if strings.Contains(message.Content, prompt) {
								foundPrompt = true
							}
						}
						if !foundPrompt {
							t.Error("legitimate private instructions missing")
						}
						if !request.Stream {
							if tc.fail || tc.malformed {
								w.WriteHeader(401)
								fmt.Fprint(w, key)
								return
							}
							fmt.Fprint(w, isolationJSON(t, map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": payload}}}}))
							return
						}
						w.Header().Set("Content-Type", "text/event-stream")
						if tc.empty {
							fmt.Fprint(w, ": heartbeat\n\ndata:\n\ndata: {}\n\n")
						}
						if tc.malformed {
							fmt.Fprint(w, "data: {broken\n\n")
							return
						}
						// Five-byte deltas split secrets and escapes across real provider frames.
						for i := 0; i < len(payload); i += 5 {
							chunk := isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": payload[i:min(i+5, len(payload))]}}}})
							if _, err := fmt.Fprintf(w, "data: %s\n\n", chunk); err != nil {
								return
							}
							w.(http.Flusher).Flush()
						}
						if tc.fail {
							fmt.Fprintf(w, "data: %s\n\n", isolationJSON(t, map[string]any{"error": map[string]string{"message": key + prompt}}))
						} else {
							fmt.Fprint(w, "data: [DONE]\n\n")
						}
					}))
					defer remote.Close()
					isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": remote.URL, "api_key": key, "custom_prompt": prompt}), owner.session, 200)
					s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(&logs, nil)), CoachDefaults: ocr.VisionConfig{HTTPClient: remote.Client()}})
					require.Equal(t, 403, do(s, "GET", "/api/v1/settings/coach", "", token).Code)
					before := f.databaseState(t)
					rec := do(s, "POST", "/api/v1/trades/"+member.trade+suffix, `{}`, token)
					require.Equal(t, 200, rec.Code)
					require.EqualValues(t, 1, calls.Load())
					noProviderSecrets(t, rec.Body.String()+fmt.Sprint(rec.Header())+logs.String(), key, prompt, url.QueryEscape(key), base64.StdEncoding.EncodeToString([]byte(key)), "unrelated-secret-token-295")
					ok := tc.safe && !tc.fail && !tc.malformed
					if strings.HasSuffix(suffix, "stream") {
						frames := sseFrames(t, rec.Body.String())
						require.NotEmpty(t, frames)
						if ok {
							require.Equal(t, []string{"note", "note", "done"}, []string{frames[0][0], frames[1][0], frames[2][0]})
							require.Len(t, frames, 3)
						} else {
							require.Equal(t, "error", frames[len(frames)-1][0])
							if tc.mixed || tc.next {
								require.Equal(t, "note", frames[0][0])
								require.Contains(t, frames[0][1], "Safe first")
							}
							for _, frame := range frames {
								if frame[0] == "note" {
									require.Contains(t, frame[1], "Safe")
								}
							}
						}
					} else {
						var result map[string]any
						require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
						if ok {
							require.Equal(t, "llm", result["source"])
						} else {
							require.Equal(t, "error", result["source"])
							require.Equal(t, "coach generation failed", result["error"])
						}
					}
					if ok {
						after := f.databaseState(t)
						delete(before, "coach_reviews")
						delete(after, "coach_reviews")
						require.Equal(t, before, after)
					} else {
						require.Equal(t, before, f.databaseState(t), "unsafe/failed generation must not save a review")
					}
					history := do(s, "GET", "/api/v1/trades/"+member.trade+"/coach/reviews", "", token)
					require.Equal(t, 200, history.Code)
					noProviderSecrets(t, history.Body.String(), key, prompt)
					before = f.databaseState(t)
					count := calls.Load()
					require.Equal(t, 404, do(s, "POST", "/api/v1/trades/"+owner.trade+suffix, `{}`, token).Code)
					require.Equal(t, count, calls.Load())
					require.Equal(t, before, f.databaseState(t))
				})
			}
		}
	}
}

func TestCoachHistoryHidesLegacyConfigurationEcho(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "owner", "ISOA", 2)
	member := f.seed(t, "member", "ISOB", 3)
	const key = "historical-private-owner-key-295"
	isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": "http://unused.invalid", "api_key": key}), owner.session, 200)
	q := store.NewForDriver(f.conn, "sqlite")
	_, err := q.CreateCoachReview(t.Context(), store.CreateCoachReviewParams{ID: "legacy-unsafe-295", UserID: member.id, TradeID: member.trade, Model: "synthetic", Notes: isolationJSON(t, []coach.Note{{Headline: "Legacy", Detail: key}})})
	require.NoError(t, err)
	for _, token := range []string{member.session, member.pat} {
		before := f.databaseState(t)
		rec := do(f.s, "GET", "/api/v1/trades/"+member.trade+"/coach/reviews", "", token)
		require.Equal(t, 200, rec.Code)
		noProviderSecrets(t, rec.Body.String(), key)
		var result struct{ Reviews []any }
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))
		require.Empty(t, result.Reviews)
		require.Equal(t, before, f.databaseState(t), "history is hidden, not rewritten")
	}
}

func TestIndependentCoachActualConnectionTermination(t *testing.T) {
	f := newIsolationFixture(t)
	_ = f.seed(t, "owner", "ISOA", 2)
	b := f.seed(t, "member", "ISOB", 3)
	for _, mode := range []string{"client-disconnect", "provider-disconnect", "provider-timeout"} {
		for _, token := range []string{b.session, b.pat} {
			t.Run(mode, func(t *testing.T) {
				const secret = "connection-secret-295"
				providerDone := make(chan struct{})
				remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					defer close(providerDone)
					if r.Header.Get("Authorization") != "Bearer "+secret {
						t.Error("configured credential not used")
					}
					_, _ = io.Copy(io.Discard, r.Body)
					w.Header().Set("Content-Type", "text/event-stream")
					w.Header().Set("X-Provider-Secret", secret)
					w.WriteHeader(200)
					chunk := isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": `{"notes":[{"tone":"tip","headline":"Partial","detail":"safe"}],`}}}})
					fmt.Fprintf(w, "data: %s\n\n", chunk)
					w.(http.Flusher).Flush()
					if mode == "provider-disconnect" {
						conn, _, err := w.(http.Hijacker).Hijack()
						if err != nil {
							t.Error(err)
							return
						}
						_ = conn.Close()
						return
					}
					select {
					case <-r.Context().Done():
					case <-time.After(5 * time.Second):
						t.Error("provider context did not terminate")
					}
				}))
				defer remote.Close()
				var logs bytes.Buffer
				q := store.NewForDriver(f.conn, "sqlite")
				cfg := ocr.VisionConfig{Enabled: true, BaseURL: remote.URL, APIKey: secret, HTTPClient: &http.Client{Timeout: 500 * time.Millisecond}}
				s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewJSONHandler(&logs, nil)), CoachDefaults: cfg})
				handlerDone := make(chan struct{})
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer close(handlerDone); s.Echo.ServeHTTP(w, r) }))
				defer server.Close()
				before := f.databaseState(t)
				req, err := http.NewRequest("POST", server.URL+"/api/v1/trades/"+b.trade+"/coach/stream", strings.NewReader(`{}`))
				require.NoError(t, err)
				req.Header.Set("Authorization", "Bearer "+token)
				req.Header.Set("Content-Type", "application/json")
				res, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
				require.NoError(t, err)
				require.Equal(t, 200, res.StatusCode)
				var output strings.Builder
				if mode == "client-disconnect" {
					sc := bufio.NewScanner(res.Body)
					for sc.Scan() {
						output.WriteString(sc.Text())
						if strings.HasPrefix(sc.Text(), "data:") {
							break
						}
					}
					require.Contains(t, output.String(), "Partial")
				} else {
					body, err := io.ReadAll(res.Body)
					require.NoError(t, err)
					output.Write(body)
					require.Contains(t, output.String(), "coach generation failed")
				}
				require.NoError(t, res.Body.Close())
				select {
				case <-handlerDone:
				case <-time.After(3 * time.Second):
					t.Fatal("API handler did not terminate")
				}
				select {
				case <-providerDone:
				case <-time.After(3 * time.Second):
					t.Fatal("provider did not terminate")
				}
				noProviderSecrets(t, output.String()+fmt.Sprint(res.Header)+logs.String(), secret, "X-Provider-Secret")
				require.Equal(t, before, f.databaseState(t))
			})
		}
	}
}
