package coach

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

func TestSecondIndependentFoldedBase64Probe(t *testing.T) {
	const key = "q7M2n9R4t6V8x1Z3"
	g, err := newOutputGuard(ocr.VisionConfig{APIKey: key})
	if err != nil {
		t.Fatal(err)
	}
	for prefix := 0; prefix < 8; prefix++ {
		plain := `{"value":"` + strings.Repeat("a", prefix) + key + `-tail"}`
		encoded := base64.StdEncoding.EncodeToString([]byte(plain))
		var folded strings.Builder
		for i := 0; i < len(encoded); i += 16 {
			if i > 0 {
				folded.WriteByte('\n')
			}
			folded.WriteString(encoded[i:min(i+16, len(encoded))])
		}
		// Standard Base64 accepts CR/LF folding and recovers the entire known credential.
		decoded, err := base64.StdEncoding.DecodeString(folded.String())
		if err != nil || !strings.Contains(string(decoded), key) {
			t.Fatal("fixture is not a recognizable echo")
		}
		if g.check(folded.String()) == nil {
			t.Errorf("recognizable folded Base64 key accepted (prefix alignment %d)", prefix)
		}
	}
}

func foldSecurityBase64(encoded, separator string, width int) string {
	var out strings.Builder
	for i := 0; i < len(encoded); i += width {
		if i > 0 {
			out.WriteString(separator)
		}
		out.WriteString(encoded[i:min(i+width, len(encoded))])
	}
	return out.String()
}

func TestCoachBase64LogicalValuesAndLimits(t *testing.T) {
	const key = "q7M2n9R4t6V8x1Z3"
	const prompt = "Owner private instructions require these distinctive limits 295."
	g, err := newOutputGuard(ocr.VisionConfig{APIKey: key, CustomPrompt: prompt})
	require.NoError(t, err)
	for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		for _, secret := range []string{key, prompt, "Authorization: Bearer " + key} {
			for _, plain := range []string{secret, `{"outer":{"value":"` + secret + `"}}`} {
				encoded := encoding.EncodeToString([]byte(plain))
				for _, separator := range []string{"\n", "\r\n", "\r"} {
					for width := 1; width <= len(encoded); width++ {
						require.ErrorIs(t, g.check(foldSecurityBase64(encoded, separator, width)), errUnsafeOutput, "fold width %d", width)
					}
				}
			}
		}
	}
	envelope := base64.StdEncoding.EncodeToString([]byte(`{"value":"` + key + `-tail"}`))
	folded := foldSecurityBase64(envelope, "\r\n", 7)
	wrapped, err := json.Marshal(map[string]any{"outer": map[string]string{"value": folded}})
	require.NoError(t, err)
	require.ErrorIs(t, g.check(string(wrapped)), errUnsafeOutput, "nested JSON with escaped newlines")
	safe := base64.StdEncoding.EncodeToString([]byte("Keep risk controlled."))
	for _, text := range []string{"prefix: " + folded + "; safe suffix", safe + " " + folded, folded + "\n" + safe, "Safe text: " + safe + "; " + key} {
		require.ErrorIs(t, g.check(text), errUnsafeOutput)
	}
	nested := `{"value":"` + key + `-tail"}`
	for range 3 {
		nested = foldSecurityBase64(base64.StdEncoding.EncodeToString([]byte(nested)), "\n", 11)
		require.ErrorIs(t, g.check(nested), errUnsafeOutput)
	}
	// Ordinary prose/math is not automatically an encoded value; valid safe text stays usable.
	for _, text := range []string{"Keep\nrisk\nsmall", "risk=2; RR=3", safe, foldSecurityBase64(safe, "\r\n", 1), safe + " " + safe, safe + "\n" + safe, base64.RawURLEncoding.EncodeToString([]byte("Keep risk controlled."))} {
		require.NoError(t, g.check(text), "legitimate output rejected")
	}
	for _, text := range []string{"YWJjZGVm===", "YWJjZGVmZ", "AA=", folded + "===", "YWJjZGVm===; " + folded, strings.Repeat("AAAA ", maxBase64Candidates+1), strings.Repeat("x", maxReviewBytes+1)} {
		require.ErrorIs(t, g.check(text), errUnsafeOutput, "malformed/over-budget encoding must fail closed")
	}
	tooDeep := "Keep risk controlled."
	for range 3 {
		tooDeep = base64.StdEncoding.EncodeToString([]byte(tooDeep))
	}
	require.NoError(t, g.check(tooDeep), "three decoding levels remain usable")
	tooDeep = base64.StdEncoding.EncodeToString([]byte(tooDeep))
	require.ErrorIs(t, g.check(tooDeep), errUnsafeOutput)
	// Near-size-limit safe content must not create excessive decoding views or token work.
	large := base64.StdEncoding.EncodeToString([]byte(strings.Repeat("!", (maxReviewBytes/4)*3-3)))
	require.NoError(t, g.check(large))
	inspected := 0
	err = inspectBase64Output("YWJjZA==\r\nZWZnaA==", new(int), actualOutput, func(value string) error { inspected++; return nil })
	require.NoError(t, err)
	require.Equal(t, 2, inspected, "separately padded values are not concatenated")
}

func TestCoachKnownConfigurationBudgets(t *testing.T) {
	_, urlErr := newOutputGuard(ocr.VisionConfig{BaseURL: strings.Repeat("x", maxReviewBytes+1)})
	require.ErrorIs(t, urlErr, errUnsafeOutput, "configuration URL budget")
	_, err := newOutputGuard(ocr.VisionConfig{APIKey: strings.Repeat("k", maxReviewBytes/2+1), CustomPrompt: strings.Repeat("p", maxReviewBytes/2)})
	require.ErrorIs(t, err, errUnsafeOutput, "combined source budget")
	var unique strings.Builder
	for i := 0; i < 4000; i++ {
		fmt.Fprintf(&unique, "%06x", i)
	}
	_, err = newOutputGuard(ocr.VisionConfig{CustomPrompt: unique.String()})
	require.ErrorIs(t, err, errUnsafeOutput, "protected fragment map must be bounded")
}
