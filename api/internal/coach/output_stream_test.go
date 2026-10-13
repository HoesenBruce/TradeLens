package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestCoachSafeNoteArrivesBeforeCompletion(t *testing.T) {
	first := make(chan Note, 1)
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "data: %s\n\n", delta(`{"notes":[{"tone":"tip","headline":"Safe first","detail":"Record the plan."}`))
		w.(http.Flusher).Flush()
		select {
		case <-release:
		case <-r.Context().Done():
			return
		}
		fmt.Fprintf(w, "data: %s\n\ndata: [DONE]\n\n", delta(`,{"tone":"tip","headline":"Safe second","detail":"Keep risk controlled."}],"next_action":"Write a journal."}`))
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	type outcome struct {
		review Review
		err    error
	}
	done := make(chan outcome, 1)
	go func() {
		rev, err := GenerateReviewStream(ctx, ocr.VisionConfig{Enabled: true, BaseURL: srv.URL, APIKey: "synthetic-transport-secret-295", CustomPrompt: "Private instruction 295.", HTTPClient: srv.Client()}, TradeContext{}, func(n Note) { first <- n })
		done <- outcome{rev, err}
	}()
	select {
	case n := <-first:
		require.Equal(t, "Safe first", n.Headline)
	case <-ctx.Done():
		t.Fatal("safe note did not arrive while provider completion was withheld")
	}
	select {
	case <-done:
		t.Fatal("review completed before provider released final content")
	default:
	}
	close(release)
	select {
	case out := <-done:
		require.NoError(t, out.err)
		require.Len(t, out.review.Notes, 2)
		require.Equal(t, "Write a journal.", out.review.NextAction)
	case <-ctx.Done():
		t.Fatal("review did not complete")
	}
}

func TestCoachStreamOutputBudgets(t *testing.T) {
	for _, wire := range []string{strings.Repeat(": keepalive\n", maxStreamBytes/10), "data: " + delta(`{"notes":[{"tone":"tip","headline":"H","detail":"`+strings.Repeat("x", maxReviewBytes)+`"}]} `) + "\n\n"} {
		t.Run(fmt.Sprint(len(wire)), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.Copy(io.Discard, r.Body); fmt.Fprint(w, wire) }))
			defer srv.Close()
			var emitted int
			_, err := GenerateReviewStream(t.Context(), ocr.VisionConfig{Enabled: true, BaseURL: srv.URL, APIKey: "key", HTTPClient: srv.Client()}, TradeContext{}, func(Note) { emitted++ })
			require.Error(t, err)
			require.Zero(t, emitted)
			require.NotContains(t, err.Error(), "xxx")
		})
	}
}

func TestCoachBlockingFallbackDoesNotDisclose(t *testing.T) {
	const key = "synthetic-fallback-secret-295"
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var request chatRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			t.Error("invalid request")
			return
		}
		if request.Stream {
			w.WriteHeader(400)
			return
		}
		body, _ := json.Marshal(map[string]any{"choices": []any{map[string]any{"message": map[string]string{"content": `{"notes":[{"tone":"tip","headline":"H","detail":"` + key + `"}],"next_action":"N"}`}}}})
		w.Write(body)
	}))
	defer srv.Close()
	var emitted int
	_, err := GenerateReviewStream(t.Context(), ocr.VisionConfig{Enabled: true, BaseURL: srv.URL, APIKey: key, HTTPClient: srv.Client()}, TradeContext{}, func(Note) { emitted++ })
	require.Error(t, err)
	require.EqualValues(t, 3, calls.Load())
	require.Zero(t, emitted)
	require.NotContains(t, err.Error(), key)
}
