package marketdata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const maxHTTPBarsBytes = 16 << 20

// ResponseProvider optionally preserves upstream metadata through the shared service.
type ResponseProvider interface {
	Provider
	FetchResponse(context.Context, Request) (Response, error)
}

// HTTPProvider consumes the application-independent Generic Bars v1 contract.
type HTTPProvider struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

func NewHTTPProvider(baseURL, apiKey string) *HTTPProvider {
	return &HTTPProvider{BaseURL: baseURL, APIKey: apiKey, Client: &http.Client{
		Timeout:       20 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}
func (p *HTTPProvider) Name() string { return "http" }
func (p *HTTPProvider) FetchBars(ctx context.Context, req Request) ([]Bar, error) {
	response, err := p.FetchResponse(ctx, req)
	return response.Bars, err
}

func (p *HTTPProvider) FetchResponse(ctx context.Context, req Request) (Response, error) {
	fail := func(message string) (Response, error) { return Response{}, errors.New("http market data: " + message) }
	u, err := url.Parse(p.BaseURL)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fail("invalid base URL")
	}
	if strings.TrimSpace(req.Symbol) == "" || req.Interval == "" || !req.From.Before(req.To) {
		return fail("invalid request")
	}
	u.Path = strings.TrimRight(u.Path, "/") + "/v1/bars"
	q := url.Values{"symbol": {chartSymbol(req)}, "instrument_type": {req.InstrumentType}, "interval": {req.Interval}, "from": {req.From.UTC().Format(time.RFC3339)}, "to": {req.To.UTC().Format(time.RFC3339)}}
	u.RawQuery = q.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return fail("invalid request")
	}
	if p.APIKey != "" {
		request.Header.Set("Authorization", "Bearer "+p.APIKey)
	}
	request.Header.Set("Accept", "application/json")
	response, err := p.Client.Do(request)
	// Never return transport errors, URLs, or response bodies: they can contain credentials.
	if err != nil {
		return fail("request failed or timed out")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fail(fmt.Sprintf("status %d", response.StatusCode))
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, maxHTTPBarsBytes+1))
	if err != nil || len(raw) > maxHTTPBarsBytes {
		return fail("response unreadable or too large")
	}
	var wire struct {
		Symbol           string `json:"symbol"`
		Interval         string `json:"interval"`
		Source           string `json:"source"`
		Timezone         string `json:"timezone"`
		AdjustmentStatus string `json:"adjustment_status"`
		Bars             *[]struct {
			Timestamp  time.Time `json:"timestamp"`
			MarketDate string    `json:"market_date"`
			Open       *float64  `json:"open"`
			High       *float64  `json:"high"`
			Low        *float64  `json:"low"`
			Close      *float64  `json:"close"`
			Volume     *float64  `json:"volume"`
			SplitRatio float64   `json:"split_ratio"`
		} `json:"bars"`
	}
	if json.Unmarshal(raw, &wire) != nil || wire.Bars == nil {
		return fail("invalid JSON or missing bars")
	}
	if wire.Symbol != chartSymbol(req) || wire.Interval != req.Interval {
		return fail("response does not match request")
	}
	var loc *time.Location
	if wire.Timezone != "" {
		loc, err = time.LoadLocation(wire.Timezone)
		if err != nil {
			return fail("invalid timezone")
		}
	}
	if wire.AdjustmentStatus == "" {
		wire.AdjustmentStatus = "unknown"
	}
	if wire.AdjustmentStatus != "unknown" && wire.AdjustmentStatus != "adjusted" && wire.AdjustmentStatus != "unadjusted" {
		return fail("invalid adjustment status")
	}
	bars := make([]Bar, 0, len(*wire.Bars))
	seen := map[int64]bool{}
	for _, b := range *wire.Bars {
		if b.Timestamp.IsZero() || b.Timestamp.Before(req.From) || !b.Timestamp.Before(req.To) || b.Open == nil || b.High == nil || b.Low == nil || b.Close == nil || b.Volume == nil {
			return fail("invalid or partial bar")
		}
		if *b.Open <= 0 || *b.Low <= 0 || *b.Close <= 0 || *b.High < *b.Open || *b.High < *b.Close || *b.Low > *b.Open || *b.Low > *b.Close || *b.Volume < 0 || b.SplitRatio < 0 || seen[b.Timestamp.Unix()] {
			return fail("invalid or duplicate bar")
		}
		seen[b.Timestamp.Unix()] = true
		date := b.MarketDate
		if loc != nil {
			expected := b.Timestamp.In(loc).Format("2006-01-02")
			if date != "" && date != expected {
				return fail("inconsistent market date")
			}
			date = expected
		} else if date != "" {
			if _, err := time.Parse("2006-01-02", date); err != nil {
				return fail("invalid market date")
			}
		}
		bars = append(bars, Bar{Time: b.Timestamp.Unix(), MarketDate: date, Open: *b.Open, High: *b.High, Low: *b.Low, Close: *b.Close, Volume: *b.Volume, SplitRatio: b.SplitRatio})
	}
	sort.Slice(bars, func(i, j int) bool { return bars[i].Time < bars[j].Time })
	return Response{Symbol: req.Symbol, Instrument: wire.Symbol, Interval: req.Interval, From: FormatTimeRFC3339(req.From), To: FormatTimeRFC3339(req.To), Provider: p.Name(), Source: wire.Source, Timezone: wire.Timezone, AdjustmentStatus: wire.AdjustmentStatus, Bars: bars}, nil
}
