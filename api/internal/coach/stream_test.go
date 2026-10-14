package coach

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestCompleteNoteObjects(t *testing.T) {
	cases := []struct {
		name string
		buf  string
		want int
	}{
		{"nothing yet", `{"no`, 0},
		{"array opened, no notes", `{"notes":[`, 0},
		{"one note still open", `{"notes":[{"tone":"tip","headline":"H"`, 0},
		{"one note closed", `{"notes":[{"tone":"tip","headline":"H","detail":"D"}`, 1},
		{
			"two closed, third open",
			`{"notes":[{"tone":"tip","headline":"A","detail":"D"},` +
				`{"tone":"warn","headline":"B","detail":"D"},{"tone":"neg"`,
			2,
		},
		{
			"array closed",
			`{"notes":[{"tone":"tip","headline":"A","detail":"D"}],"next_action":"go"}`,
			1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Len(t, completeNoteObjects(tc.buf), tc.want)
		})
	}
}

// Braces and brackets inside a string must not move the nesting depth, or a
// headline containing punctuation would desynchronise the whole scan.
func TestCompleteNoteObjectsIgnoresBracesInStrings(t *testing.T) {
	buf := `{"notes":[{"tone":"tip","headline":"use {} and [] carefully","detail":"D"}`
	objs := completeNoteObjects(buf)
	require.Len(t, objs, 1)
	require.Contains(t, objs[0], "use {} and [] carefully")
}

func TestCompleteNoteObjectsHandlesEscapedQuotes(t *testing.T) {
	buf := `{"notes":[{"tone":"tip","headline":"say \"hi\"","detail":"D"},{"tone":"warn"`
	objs := completeNoteObjects(buf)
	require.Len(t, objs, 1)
	require.Contains(t, objs[0], `\"hi\"`)
}

// A note whose detail happens to contain the word notes must not be mistaken
// for the start of the array.
func TestCompleteNoteObjectsAnchorsOnTheArrayKey(t *testing.T) {
	buf := `{"next_action":"review notes","notes":[{"tone":"tip","headline":"H","detail":"D"}`
	require.Len(t, completeNoteObjects(buf), 1)
}

func TestEmitNewNotesEmitsEachNoteOnce(t *testing.T) {
	var got []Note
	onNote := func(n Note) { got = append(got, n) }

	buf := `{"notes":[{"tone":"tip","headline":"A","detail":"D"}`
	emitted := emitNewNotes(buf, 0, onNote)
	require.Equal(t, 1, emitted)
	require.Len(t, got, 1)
	require.Equal(t, "A", got[0].Headline)

	// The same prefix again must not re-emit.
	emitted = emitNewNotes(buf, emitted, onNote)
	require.Equal(t, 1, emitted)
	require.Len(t, got, 1)

	buf += `,{"tone":"warn","headline":"B","detail":"D"}`
	emitted = emitNewNotes(buf, emitted, onNote)
	require.Equal(t, 2, emitted)
	require.Len(t, got, 2)
	require.Equal(t, "B", got[1].Headline)
	require.Equal(t, "warn", got[1].Tone)
	require.Equal(t, 2, got[1].Priority)
}

// sseServer replays frames as an OpenAI-compatible stream.
func sseServer(t *testing.T, frames []string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		fl, ok := w.(http.Flusher)
		require.True(t, ok)
		for _, f := range frames {
			fmt.Fprintf(w, "data: %s\n\n", f)
			fl.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		fl.Flush()
	}))
}

// delta wraps a content fragment in one OpenAI-compatible stream chunk.
func delta(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return `{"choices":[{"delta":{"content":` + string(b) + `}}]}`
}

func TestGenerateReviewStreamUnavailable(t *testing.T) {
	_, err := GenerateReviewStream(
		context.Background(), ocr.VisionConfig{}, TradeContext{}, nil,
	)
	require.ErrorIs(t, err, ErrUnavailable)
}
