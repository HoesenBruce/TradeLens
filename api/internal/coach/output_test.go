package coach

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestCoachOutputGuardRepresentationsAndBounds(t *testing.T) {
	const key = "synthetic+owner-secret-key/295"
	const prompt = "Owner private instructions contain this distinctive sentence.\nHidden private line 295"
	cfg := ocr.VisionConfig{APIKey: key, CustomPrompt: prompt, BaseURL: "https://url-user:url-password-295@example.invalid/?token=url-token-295"}
	g, err := newOutputGuard(cfg)
	require.NoError(t, err)
	var escaped strings.Builder
	pieces := map[string]string{}
	for i := 0; i < len(key); i += 3 {
		pieces[fmt.Sprintf("p%02d", i)] = key[i:min(i+3, len(key))]
	}
	structuredPieces, err := json.Marshal(pieces)
	require.NoError(t, err)
	for _, r := range key {
		fmt.Fprintf(&escaped, `\u%04x`, r)
	}
	for name, text := range map[string]string{
		"raw":               key,
		"multiline":         strings.Join(strings.Split(key, ""), "\n"),
		"url":               url.QueryEscape(url.QueryEscape(key)),
		"unicode":           escaped.String(),
		"html":              strings.ReplaceAll(key, "+", "&#43;"),
		"base64 key":        base64.StdEncoding.EncodeToString([]byte(key)),
		"base64 envelope":   base64.StdEncoding.EncodeToString([]byte(`{"message":"` + key + `"}`)),
		"hex":               hex.EncodeToString([]byte(key)),
		"structured":        `{"secret":{"value":"` + key + `"}}`,
		"structured pieces": string(structuredPieces),
		"prompt":            prompt,
		"prompt line":       "Hidden private line 295",
		"prompt fragment":   "private instructions contain this distinctive sentence",
		"bearer":            "Bearer unknown-credential-295",
		"short bearer":      "Bearer abc",
		"short basic":       "Basic YTo=",
		"authorization":     "AUTHORI\nZATION = anything",
		"url password":      "url-password-295",
		"url token":         "url-token-295",
	} {
		t.Run(name, func(t *testing.T) { require.ErrorIs(t, g.check(text), errUnsafeOutput) })
	}
	require.NoError(t, g.check("Keep risk under control. Write a journal."))
	require.ErrorIs(t, g.check(strings.Repeat("x", maxReviewBytes+1)), errUnsafeOutput)
	require.ErrorIs(t, g.check(string([]byte{0xff})), errUnsafeOutput)
	tooDeep := "safe text with spaces"
	for range 5 {
		tooDeep = base64.StdEncoding.EncodeToString([]byte(tooDeep))
	}
	require.ErrorIs(t, g.check(tooDeep), errUnsafeOutput)
	tooNested := strings.Repeat("[", 18) + `"safe"` + strings.Repeat("]", 18)
	require.ErrorIs(t, g.check(tooNested), errUnsafeOutput)
	_, err = newOutputGuard(ocr.VisionConfig{CustomPrompt: strings.Repeat("x", maxReviewBytes+1)})
	require.ErrorIs(t, err, errUnsafeOutput)
}

func TestCoachModelMessagesOmitTransportCredential(t *testing.T) {
	cfg := ocr.VisionConfig{APIKey: "synthetic-transport-key-295", CustomPrompt: "Use risk limits; key synthetic-transport-key-295 is not an instruction."}
	require.NotContains(t, systemPrompt(cfg), cfg.APIKey)
	require.Contains(t, systemPrompt(cfg), "Use risk limits")
	guard, err := newOutputGuard(cfg)
	require.NoError(t, err)
	require.ErrorIs(t, guard.check(systemPrompt(cfg)), errUnsafeOutput)
	require.NotContains(t, withoutTransportCredentials("legacy journal key "+cfg.APIKey, cfg), cfg.APIKey)
	require.Equal(t, "Keep risk small.", withoutTransportCredentials("Keep risk small.", ocr.VisionConfig{APIKey: "sk"}))
	text := "Keep risk controlled.\nAuthorization: Bearer other-owner-token-295\nJournal the exit."
	require.Equal(t, "Keep risk controlled.\n[credential omitted]\nJournal the exit.", withoutTransportCredentials(text, cfg))
}
