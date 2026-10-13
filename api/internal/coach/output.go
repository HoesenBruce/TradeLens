package coach

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/tradermemos/api/internal/ocr"
)

const maxReviewBytes = 64 << 10
const maxStreamBytes = 4 << 20
const maxInspectionViews = 128
const maxInspectionBytes = 4 * maxReviewBytes
const maxBase64Candidates = 1024
const maxProtectedFragments = 16384

var errUnsafeOutput = errors.New("coach output rejected")
var credentialText = regexp.MustCompile(`(?i)(authorization\s*["']?\s*[:=]|bearer\s+[a-z0-9+/_=.-]+|basic\s+(?:[a-z0-9+/_-]*=|[a-z0-9+/_-]{8,})|(?:api[_ -]?key|access[_ -]?token|password|secret)\s*["']?\s*[:=])`)
var unicodeEscape = regexp.MustCompile(`(?:\\u[0-9a-fA-F]{4})+`)

func transportCredentials(cfg ocr.VisionConfig) []string {
	values := []string{strings.TrimSpace(cfg.APIKey)}
	if u, err := url.Parse(cfg.BaseURL); err == nil {
		if u.User != nil {
			password, _ := u.User.Password()
			values = append(values, u.User.Username(), password)
		}
		for name, queryValues := range u.Query() {
			switch strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(name)) {
			case "apikey", "key", "token", "accesstoken", "secret", "clientsecret", "password", "authorization", "signature", "sig":
				values = append(values, queryValues...)
			}
		}
	}
	return values
}

func withoutTransportCredentials(text string, cfg ocr.VisionConfig) string {
	for _, key := range transportCredentials(cfg) {
		if key == "" {
			continue
		}
		if len(key) >= 8 {
			text = strings.ReplaceAll(text, key, "[credential omitted]")
		} else {
			text = regexp.MustCompile(`\b`+regexp.QuoteMeta(key)+`\b`).ReplaceAllString(text, "[credential omitted]")
		}
	}
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if credentialText.MatchString(line) {
			lines[i] = "[credential omitted]"
		}
	}
	return strings.Join(lines, "\n")
}

// A complete bounded note is the streaming unit, never an unvalidated text delta.
// shortcut: recognisable echoes only; arbitrary ciphers/paraphrases require a trusted structured provider.
type outputGuard struct {
	values    []string
	literals  []string
	fragments map[int]map[string]bool
}

func newOutputGuard(cfg ocr.VisionConfig) (*outputGuard, error) {
	if len(cfg.BaseURL) > maxReviewBytes || len(cfg.APIKey) > maxReviewBytes || len(cfg.CustomPrompt) > maxReviewBytes {
		return nil, errUnsafeOutput
	}
	g := &outputGuard{fragments: make(map[int]map[string]bool)}
	sourceBudget, fragmentCount := maxReviewBytes, 0
	items := []struct {
		value  string
		window int
	}{{strings.TrimSpace(cfg.CustomPrompt), 32}}
	if cfg.CustomPrompt != "" && systemPrompt(cfg) != strings.TrimSpace(cfg.CustomPrompt) {
		items = append(items, struct {
			value  string
			window int
		}{systemPrompt(cfg), 32})
	}
	for _, credential := range transportCredentials(cfg) {
		items = append(items, struct {
			value  string
			window int
		}{credential, 12})
	}
	for _, item := range items {
		if len(item.value) > sourceBudget {
			return nil, errUnsafeOutput
		}
		sourceBudget -= len(item.value)
		if item.value == "" {
			continue
		}
		if outputCanonical(item.value) == "" {
			g.literals = append(g.literals, item.value)
		}
		for _, value := range []string{item.value, base64.StdEncoding.EncodeToString([]byte(item.value)), base64.RawURLEncoding.EncodeToString([]byte(item.value)), hex.EncodeToString([]byte(item.value))} {
			value = outputCanonical(value)
			if value == "" {
				continue
			}
			g.values = append(g.values, value)
			if len(value) < 2*item.window {
				continue
			}
			if g.fragments[item.window] == nil {
				g.fragments[item.window] = make(map[string]bool)
			}
			for i := 0; i+item.window <= len(value); i++ {
				fragment := value[i : i+item.window]
				if !g.fragments[item.window][fragment] {
					if fragmentCount >= maxProtectedFragments {
						return nil, errUnsafeOutput
					}
					g.fragments[item.window][fragment] = true
					fragmentCount++
				}
			}
		}
		// A short private line is identifiable even inside a longer custom prompt.
		if item.window == 32 {
			for _, line := range strings.Split(item.value, "\n") {
				if value := outputCanonical(line); len(value) >= 8 {
					g.values = append(g.values, value)
				}
			}
		}
		if len(g.values) > 128 {
			return nil, errUnsafeOutput
		}
	}
	return g, nil
}

func outputCanonical(text string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, text)
}

// The zero value and unrecognized origins are strict; only internal joins get an exception.
type outputOrigin uint8

const (
	actualOutput outputOrigin = iota
	decodedContent
	syntheticInspection
)

type inspectionContent struct {
	text   string
	origin outputOrigin
}

func (g *outputGuard) check(text string) error {
	return g.checkText(inspectionContent{text, actualOutput})
}

func (g *outputGuard) checkText(initial inspectionContent) error {
	// The decoding budget bounds both memory and work on adversarial output.
	type view struct {
		inspectionContent
		depth int
	}
	queue := []view{{initial, 0}}
	scheduled := map[inspectionContent]bool{initial: true}
	budget := maxInspectionBytes - len(initial.text)
	candidates := 0
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		if len(v.text) > maxReviewBytes || !utf8.ValidString(v.text) || strings.ContainsRune(v.text, utf8.RuneError) {
			return errUnsafeOutput
		}
		canonical := outputCanonical(v.text)
		for _, literal := range g.literals {
			if strings.Contains(v.text, literal) {
				return errUnsafeOutput
			}
		}
		if credentialText.MatchString(v.text) || strings.Contains(canonical, "authorization") {
			return errUnsafeOutput
		}
		for _, secret := range g.values {
			if len(secret) >= 8 {
				if strings.Contains(canonical, secret) {
					return errUnsafeOutput
				}
			} else {
				// Short local-provider keys must match a whole token, not a letter in "risk".
				for _, word := range strings.FieldsFunc(v.text, func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }) {
					if outputCanonical(word) == secret {
						return errUnsafeOutput
					}
				}
			}
		}
		for width, fragments := range g.fragments {
			for i := 0; i+width <= len(canonical); i++ {
				if fragments[canonical[i:i+width]] {
					return errUnsafeOutput
				}
			}
		}
		add := func(c inspectionContent) error {
			decoded := c.text
			if scheduled[c] {
				return nil
			}
			if v.depth >= 3 || len(decoded) > maxReviewBytes || len(scheduled) >= maxInspectionViews || len(decoded) > budget {
				return errUnsafeOutput
			}
			budget -= len(decoded)
			scheduled[c] = true
			queue = append(queue, view{c, v.depth + 1})
			return nil
		}
		// Identity transforms create no new decoded value, especially for synthetic joins.
		addDecoded := func(text string) error {
			if text == v.text {
				return nil
			}
			return add(inspectionContent{text, decodedContent})
		}
		if err := addDecoded(html.UnescapeString(v.text)); err != nil {
			return err
		}
		if decoded, err := url.QueryUnescape(v.text); err == nil {
			if err := addDecoded(decoded); err != nil {
				return err
			}
		}
		invalidEscape := false
		decoded := unicodeEscape.ReplaceAllStringFunc(v.text, func(s string) string {
			var value string
			if json.Unmarshal([]byte(`"`+s+`"`), &value) != nil {
				invalidEscape = true
			}
			return value
		})
		if invalidEscape {
			return errUnsafeOutput
		}
		if err := addDecoded(decoded); err != nil {
			return err
		}
		parts, err := structuredOutputText(v.text)
		if err != nil {
			return err
		}
		for _, part := range parts {
			if err := add(part); err != nil {
				return err
			}
		}
		if err := inspectBase64Output(v.text, &candidates, v.origin, func(decoded string) error { return add(inspectionContent{decoded, decodedContent}) }); err != nil {
			return err
		}
	}
	return nil
}

// CR/LF belongs to the encoded value; decode it whole, never as separate lines.
func inspectBase64Output(text string, candidates *int, origin outputOrigin, add func(string) error) error {
	alphabet := func(c byte) bool {
		return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || strings.ContainsRune("+/_-=", rune(c))
	}
	for i := 0; i < len(text); {
		if !alphabet(text[i]) {
			i++
			continue
		}
		start := i
		padded, breakAfterPadding := false, false
		for i < len(text) {
			c := text[i]
			if c == '\r' || c == '\n' {
				breakAfterPadding = padded
			} else if !alphabet(c) || breakAfterPadding && c != '=' {
				break
			}
			padded = padded || c == '='
			i++
		}
		value := strings.TrimRight(text[start:i], "\r\n")
		marked := strings.HasSuffix(value, "=") || strings.Contains(value, "==")
		if len(value) < 4 && !marked {
			continue
		}
		*candidates++
		if *candidates > maxBase64Candidates {
			return errUnsafeOutput
		}
		textDecoded := false
		partialText := false
		for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
			decoded, err := encoding.DecodeString(value)
			partialText = partialText || err != nil && len(decoded) >= 4 && utf8.Valid(decoded)
			if err != nil && !utf8.Valid(decoded) {
				// Inspect recovered text even when corrupt terminal bytes make the whole partial result invalid.
				prefix := 0
				for prefix < len(decoded) {
					r, size := utf8.DecodeRune(decoded[prefix:])
					if r == utf8.RuneError && size == 1 {
						break
					}
					prefix += size
				}
				if prefix >= 4 {
					if err := add(string(decoded[:prefix])); err != nil {
						return err
					}
				}
			}
			if err == nil && utf8.Valid(decoded) {
				textDecoded = true
				if err := add(string(decoded)); err != nil {
					return err
				}
				break
			}
		}
		// Padding identifies encoded data; malformed/binary values cannot be approved as text.
		if origin != syntheticInspection && !textDecoded && (marked || partialText) {
			return errUnsafeOutput
		}
	}
	return nil
}

// Preserve JSON source order while checking values without intervening property names.
func structuredOutputText(text string) ([]inspectionContent, error) {
	if !json.Valid([]byte(text)) {
		return nil, nil
	}
	d := json.NewDecoder(strings.NewReader(text))
	d.UseNumber()
	var keys, values strings.Builder
	var parts []inspectionContent
	leaf := func(text string) error {
		if len(parts) >= maxInspectionViews {
			return errUnsafeOutput
		}
		parts = append(parts, inspectionContent{text, decodedContent})
		return nil
	}
	var visit func(int) error
	visit = func(depth int) error {
		if depth > 16 {
			return errUnsafeOutput
		}
		token, err := d.Token()
		if err != nil {
			return errUnsafeOutput
		}
		switch value := token.(type) {
		case string:
			if err := leaf(value); err != nil {
				return err
			}
			values.WriteString(value)
		case json.Number:
			if err := leaf(value.String()); err != nil {
				return err
			}
			values.WriteString(value.String())
		case bool:
			if err := leaf(fmt.Sprint(value)); err != nil {
				return err
			}
			if value {
				values.WriteString("true")
			} else {
				values.WriteString("false")
			}
		case nil:
			if err := leaf("null"); err != nil {
				return err
			}
			values.WriteString("null")
		case json.Delim:
			for d.More() {
				if value == '{' {
					key, err := d.Token()
					if err != nil {
						return errUnsafeOutput
					}
					name, ok := key.(string)
					if !ok {
						return errUnsafeOutput
					}
					if err := leaf(name); err != nil {
						return err
					}
					keys.WriteString(name)
				}
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			if _, err := d.Token(); err != nil {
				return errUnsafeOutput
			}
		}
		return nil
	}
	if err := visit(0); err != nil {
		return nil, err
	}
	return append(parts,
		inspectionContent{keys.String(), syntheticInspection},
		inspectionContent{values.String(), syntheticInspection}), nil
}

func (g *outputGuard) review(review Review) error {
	if len(review.Notes) > maxNotes {
		return errUnsafeOutput
	}
	var text strings.Builder
	for _, n := range review.Notes {
		if len(n.Headline)+len(n.Detail) > maxReviewBytes-text.Len() {
			return errUnsafeOutput
		}
		if err := g.note(n); err != nil {
			return err
		}
		text.WriteString(n.Headline)
		text.WriteString(n.Detail)
	}
	if err := g.check(review.NextAction); err != nil {
		return err
	}
	text.WriteString(review.NextAction)
	return g.checkText(inspectionContent{text.String(), syntheticInspection})
}

func (g *outputGuard) note(n Note) error {
	for _, text := range []string{n.Headline, n.Detail} {
		if err := g.check(text); err != nil {
			return err
		}
	}
	return g.checkText(inspectionContent{n.Headline + n.Detail, syntheticInspection})
}

// ValidateReviewOutput also protects stored history and blocking/fallback responses.
func ValidateReviewOutput(cfg ocr.VisionConfig, review Review, model string) error {
	g, err := newOutputGuard(cfg)
	if err != nil {
		return err
	}
	if err := g.check(model); err != nil {
		return err
	}
	return g.review(review)
}
