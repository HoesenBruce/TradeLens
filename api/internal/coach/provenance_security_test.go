package coach

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/ocr"
)

// Reconstructed from the published third independent review.
func TestThirdIndependentMalformedEscapedJSON(t *testing.T) {
	key := fmt.Sprintf("SYNTH_%x", sha256.Sum256([]byte("FIXTURE_CONFIG_A")))[:26]
	g, err := newOutputGuard(ocr.VisionConfig{APIKey: key})
	require.NoError(t, err)
	encoded := base64.RawStdEncoding.EncodeToString([]byte(`{"value":"` + key + `-tail"}`))
	for len(encoded)%4 != 0 {
		encoded += "A"
	}
	for _, suffix := range []string{"X", "XXXXX", "===="} {
		t.Run(fmt.Sprint(len(suffix)), func(t *testing.T) {
			folded := foldSecurityBase64(encoded+suffix, "\n", 16)
			decoded, err := base64.RawStdEncoding.DecodeString(folded)
			require.True(t, err != nil && strings.Contains(string(decoded), key), "fixture oracle")
			wrapped, err := json.Marshal(map[string]string{"data": folded})
			require.NoError(t, err)
			if g.check(string(wrapped)) == nil {
				t.Error("FIXTURE_CONFIG_A accepted in decoded JSON")
			}
		})
	}
}

func TestThirdIndependentLegitimateOutputs(t *testing.T) {
	g, err := newOutputGuard(ocr.VisionConfig{APIKey: fmt.Sprintf("%x", sha256.Sum256([]byte("FIXTURE_CONFIG_A")))})
	require.NoError(t, err)
	safe := base64.StdEncoding.EncodeToString([]byte("Keep risk controlled."))
	wrapped, err := json.Marshal(map[string]string{"first": safe, "second": safe})
	require.NoError(t, err)
	for _, review := range []Review{
		{Notes: []Note{{Headline: safe, Detail: safe}}, NextAction: safe},
		{Notes: []Note{{Headline: "Safe review", Detail: string(wrapped)}}},
		{Notes: []Note{{Headline: "Safe review", Detail: "Keep\nrisk\nsmall"}, {Headline: "交易复盘", Detail: "リスク管理を確認。"}}},
		{Notes: []Note{{Headline: "Safe review", Detail: strings.Repeat("Keep risk small. ", 150)}}},
	} {
		require.NoError(t, g.review(review), "legitimate output rejected")
	}
}

func TestCoachOutputProvenanceStrictness(t *testing.T) {
	g, err := newOutputGuard(ocr.VisionConfig{})
	require.NoError(t, err)
	malformed := "YWJjZGVmZ"
	for _, origin := range []outputOrigin{actualOutput, decodedContent, outputOrigin(255)} {
		require.ErrorIs(t, g.checkText(inspectionContent{malformed, origin}), errUnsafeOutput, "actual/decoded/unknown must be strict")
	}
	require.NoError(t, g.checkText(inspectionContent{malformed, syntheticInspection}), "non-emittable join artifact")
	// Real transforms from a joined view must always regain strict provenance.
	for _, text := range []string{
		"%59WJjZGVmZ",
		"&#89;WJjZGVmZ",
		`\u0059WJjZGVmZ`,
		base64.StdEncoding.EncodeToString([]byte(malformed)),
		`{"value":"YWJjZGVmZ"}`,
		`{"YWJjZGVmZ":"safe prose."}`,
		`["YWJjZGVmZ","safe prose."]`,
	} {
		require.ErrorIs(t, g.checkText(inspectionContent{text, syntheticInspection}), errUnsafeOutput, "decoded content must not inherit synthetic origin")
	}
	parts, err := structuredOutputText(`{"first":"Keep risk small.","second":"Trade carefully."}`)
	require.NoError(t, err)
	require.Len(t, parts, 6)
	for _, part := range parts[:4] {
		require.Equal(t, decodedContent, part.origin)
	}
	for _, part := range parts[4:] {
		require.Equal(t, syntheticInspection, part.origin)
	}
}

func TestCoachProvenanceResourceLimits(t *testing.T) {
	g, err := newOutputGuard(ocr.VisionConfig{})
	require.NoError(t, err)
	for _, text := range []string{
		strings.Repeat("[", 17) + `"safe prose."` + strings.Repeat("]", 17),
		"[" + strings.Repeat(`"safe prose.",`, maxInspectionViews) + `"safe prose."]`,
		strings.Repeat("!", maxReviewBytes+1),
		strings.Repeat("AAAA ", maxBase64Candidates+1),
		string([]byte{0xff, 0xfe}),
		`"\ud800"`,
	} {
		require.ErrorIs(t, g.check(text), errUnsafeOutput, "inspection must fail closed at limits")
	}
	// Separate decoded fields consume a shared scheduling budget before enqueue.
	fields := map[string]string{}
	for i := 0; i < 90; i++ {
		fields[fmt.Sprint(i)] = strings.Repeat("!", 600) + fmt.Sprint(i)
	}
	wrapped, err := json.Marshal(fields)
	require.NoError(t, err)
	require.ErrorIs(t, g.check(string(wrapped)), errUnsafeOutput, "scheduled view limit")
}
