package coach

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestFormatTradeContext(t *testing.T) {
	net := 100.0
	r := 2.0
	out := FormatTradeContext(TradeContext{
		Symbol:    "AAPL",
		Direction: "long",
		Status:    "closed",
		OpenedAt:  time.Date(2026, 7, 1, 14, 0, 0, 0, time.UTC),
		NetPnl:    &net,
		Currency:  "USD",
		RMultiple: &r,
		Notes:     "entry: breakout",
		TagNames:  []string{"orb"},
		Fills: []FillContext{{
			Side: "buy", Quantity: 10, Price: 190, ExecutedAt: time.Date(2026, 7, 1, 14, 0, 0, 0, time.UTC),
		}},
	})
	for _, want := range []string{"AAPL", "Net P&L: 100", "R-multiple: 2", "entry: breakout", "orb", "buy 10 @ 190"} {
		require.Contains(t, out, want)
	}
}

func TestResponseFormatPromptRequiresNextAction(t *testing.T) {
	require.Contains(t, responseFormatPrompt, "next_action")
	require.Contains(t, responseFormatPrompt, "Required.")
}

func TestGenerateReview_Unavailable(t *testing.T) {
	_, err := GenerateReview(context.Background(), ocr.VisionConfig{}, TradeContext{})
	require.ErrorIs(t, err, ErrUnavailable)
}

func TestNormalizeTone(t *testing.T) {
	cases := map[string]string{
		"NEG": "neg", "warning": "warn", "Strength": "pos", "habit": "tip", "": "tip",
	}
	for in, want := range cases {
		require.Equal(t, want, normalizeTone(in), "input %q", in)
	}
}
