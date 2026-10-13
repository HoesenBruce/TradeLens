package coach

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
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

var errUnsafeOutput = errors.New("coach output rejected")
var credentialText = regexp.MustCompile(`(?i)(authorization\s*["']?\s*[:=]|bearer\s+[a-z0-9+/_=.-]+|basic\s+(?:[a-z0-9+/_-]*=|[a-z0-9+/_-]{8,})|(?:api[_ -]?key|access[_ -]?token|password|secret)\s*["']?\s*[:=])`)
var encodedToken = regexp.MustCompile(`[A-Za-z0-9+/_=-]{4,}`)
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
	g := &outputGuard{fragments: make(map[int]map[string]bool)}
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
		if len(item.value) > maxReviewBytes {
			return nil, errUnsafeOutput
		}
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
				g.fragments[item.window][value[i:i+item.window]] = true
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

func (g *outputGuard) check(text string) error {
	// The decoding budget bounds both memory and work on adversarial output.
	type view struct {
		text  string
		depth int
	}
	queue := []view{{text, 0}}
	seen := make(map[string]bool)
	budget := 4 * maxReviewBytes
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		if seen[v.text] {
			continue
		}
		seen[v.text] = true
		budget -= len(v.text)
		if len(v.text) > maxReviewBytes || budget < 0 || len(seen) > 128 || !utf8.ValidString(v.text) || strings.ContainsRune(v.text, utf8.RuneError) {
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
		add := func(decoded string) error {
			if decoded == v.text || seen[decoded] {
				return nil
			}
			if v.depth >= 3 {
				return errUnsafeOutput
			}
			queue = append(queue, view{decoded, v.depth + 1})
			return nil
		}
		if err := add(html.UnescapeString(v.text)); err != nil {
			return err
		}
		if decoded, err := url.QueryUnescape(v.text); err == nil {
			if err := add(decoded); err != nil {
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
		if err := add(decoded); err != nil {
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
		for _, token := range encodedToken.FindAllString(v.text, -1) {
			for _, encoding := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
				decoded, err := encoding.DecodeString(token)
				if err == nil && utf8.Valid(decoded) {
					if err := add(string(decoded)); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// Preserve JSON source order while checking values without intervening property names.
func structuredOutputText(text string) ([]string, error) {
	if !json.Valid([]byte(text)) {
		return nil, nil
	}
	d := json.NewDecoder(strings.NewReader(text))
	d.UseNumber()
	var keys, values strings.Builder
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
			values.WriteString(value)
		case json.Number:
			values.WriteString(value.String())
		case bool:
			if value {
				values.WriteString("true")
			} else {
				values.WriteString("false")
			}
		case nil:
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
	return []string{keys.String(), values.String()}, nil
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
	return g.check(text.String())
}

func (g *outputGuard) note(n Note) error {
	for _, text := range []string{n.Headline, n.Detail, n.Headline + n.Detail} {
		if err := g.check(text); err != nil {
			return err
		}
	}
	return nil
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
