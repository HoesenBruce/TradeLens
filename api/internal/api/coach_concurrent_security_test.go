package api_test

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/api"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/ocr"
	"github.com/tradermemos/api/internal/store"
	"github.com/tradermemos/api/internal/trades"
)

func TestSecondIndependentConcurrentRetryStreaming(t *testing.T) {
	f := newIsolationFixture(t)
	owner := f.seed(t, "owner", "OWN295", 2)
	member := f.seed(t, "member", "MEM295", 3)
	other := f.seed(t, "other", "OTH295", 4)
	participants := []struct{ token, trade, user string }{{member.session, member.trade, member.id}, {other.pat, other.trade, other.id}}
	var releaseOnce sync.Once
	const key = "concurrent-owner-credential-295"
	const prompt = "Owner private instructions for concurrent review 295."
	var calls atomic.Int64
	release := make(chan struct{})
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := calls.Add(1)
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer "+key || !strings.Contains(string(body), prompt) {
			t.Error("configured transport/private prompt not received")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		partial := `{"notes":[{"tone":"tip","headline":"Safe preview","detail":"Keep risk controlled."}],`
		fmt.Fprintf(w, "data: %s\n\n", isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": partial}}}}))
		w.(http.Flusher).Flush()
		if n <= 2 {
			fmt.Fprint(w, `data: {"error":{"message":"synthetic upstream failure"}}`+"\n\n")
			return
		}
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", isolationJSON(t, map[string]any{"choices": []any{map[string]any{"delta": map[string]string{"content": `"next_action":"Record the next plan."}`}}}}))
	}))
	defer remote.Close()
	isolationObject(t, f.s, "PUT", "/settings/coach", isolationJSON(t, map[string]any{"enabled": true, "base_url": remote.URL, "api_key": key, "custom_prompt": prompt}), owner.session, 200)
	q := store.NewForDriver(f.conn, "sqlite")
	s := api.New(api.Deps{JWT: auth.NewJWT("disposable-isolation-test-secret"), Store: q, Trades: trades.NewService(q), Logger: slog.New(slog.NewTextHandler(io.Discard, nil)), CoachDefaults: ocr.VisionConfig{HTTPClient: remote.Client()}})
	before := f.databaseState(t)
	for _, participant := range participants {
		rec := do(s, "POST", "/api/v1/trades/"+participant.trade+"/coach/stream", `{}`, participant.token)
		frames := sseFrames(t, rec.Body.String())
		require.Len(t, frames, 2)
		require.Equal(t, "note", frames[0][0])
		require.Equal(t, "error", frames[1][0])
		noProviderSecrets(t, rec.Body.String()+fmt.Sprint(rec.Header()), key, prompt)
	}
	require.Equal(t, before, f.databaseState(t), "failed attempts do not persist")
	server := httptest.NewServer(s.Echo)
	defer server.Close()
	first := make(chan bool, 2)
	type result struct {
		body, headers string
		err           error
	}
	done := make(chan result, 2)
	for _, participant := range participants {
		go func(token, trade string) {
			req, _ := http.NewRequest("POST", server.URL+"/api/v1/trades/"+trade+"/coach/stream", strings.NewReader(`{}`))
			req.Header.Set("Authorization", "Bearer "+token)
			req.Header.Set("Content-Type", "application/json")
			res, err := (&http.Client{Timeout: 8 * time.Second}).Do(req)
			if err != nil {
				done <- result{err: err}
				return
			}
			defer res.Body.Close()
			var full strings.Builder
			reader := bufio.NewReader(res.Body)
			seen := false
			for {
				line, err := reader.ReadString('\n')
				full.WriteString(line)
				if strings.HasPrefix(line, "data:") {
					seen = strings.Contains(line, "Safe preview")
					break
				}
				if err != nil {
					done <- result{err: err}
					return
				}
			}
			first <- seen
			rest, err := io.ReadAll(reader)
			full.Write(rest)
			done <- result{body: full.String(), headers: fmt.Sprint(res.Header), err: err}
		}(participant.token, participant.trade)
	}
	// Even a test failure must release the provider before server cleanup.
	defer releaseOnce.Do(func() { close(release) })
	for range 2 {
		select {
		case ok := <-first:
			require.True(t, ok)
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent safe preview did not arrive")
		}
	}
	select {
	case <-done:
		t.Fatal("completed before provider released final content")
	default:
	}
	// Release completion only after both clients have received a preview.
	releaseOnce.Do(func() { close(release) })
	for range 2 {
		select {
		case out := <-done:
			require.NoError(t, out.err)
			noProviderSecrets(t, out.body+out.headers, key, prompt)
			frames := sseFrames(t, out.body)
			require.Len(t, frames, 2)
			require.Equal(t, "note", frames[0][0])
			require.Equal(t, "done", frames[1][0])
			require.Contains(t, frames[1][1], "Record the next plan.")
		case <-time.After(5 * time.Second):
			t.Fatal("retry did not finish")
		}
	}
	for _, participant := range participants {
		var count int
		require.NoError(t, f.conn.QueryRow("SELECT COUNT(*) FROM coach_reviews WHERE user_id=? AND trade_id=?", participant.user, participant.trade).Scan(&count))
		require.Equal(t, 1, count, "each user persists only their own review")
	}
	after := f.databaseState(t)
	delete(before, "coach_reviews")
	delete(after, "coach_reviews")
	require.Equal(t, before, after)
	require.EqualValues(t, 4, calls.Load())
}
