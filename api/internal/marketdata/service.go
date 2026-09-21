package marketdata

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tradermemos/api/internal/store"
	"golang.org/x/sync/singleflight"
)

const DefaultDailyMarketDataLookbackDays = 365

// Service resolves bar requests with in-memory LRU, SQLite cache, and provider fetch.
type Service struct {
	Store    store.Querier
	Provider Provider
	mem      *memCache
	group    singleflight.Group
	fxMem    *fxCache
	fxGroup  singleflight.Group
}

func NewService(q store.Querier, provider Provider) *Service {
	return &Service{
		Store:    q,
		Provider: provider,
		// 512 windows: a bar-replay session walks many distinct (symbol,
		// interval, window) keys and would thrash a smaller cache.
		mem:   newMemCache(512),
		fxMem: newFxCache(),
	}
}

func (s *Service) GetBars(ctx context.Context, req Request) (Response, error) {
	if s.Provider == nil {
		return Response{}, errors.New("market data provider not configured")
	}
	if provider, ok := s.Provider.(ResponseProvider); ok {
		// Preserve response metadata rather than passing it through the legacy bars-only cache.
		return provider.FetchResponse(ctx, req)
	}
	key := CacheKey(req)
	if bars, ok := s.mem.get(key); ok {
		return responseFor(req, s.Provider.Name(), true, bars), nil
	}

	v, err, _ := s.group.Do(key, func() (any, error) {
		return s.fetchBars(ctx, req, key)
	})
	if err != nil {
		return Response{}, err
	}
	return v.(Response), nil
}

// GetTransactionBars returns broad shared daily coverage while preserving the
// explicit request window in Response.From/To for the chart's initial viewport.
func (s *Service) GetTransactionBars(ctx context.Context, req Request, earliest time.Time) (Response, error) {
	if s.Provider == nil {
		return Response{}, errors.New("market data provider not configured")
	}
	if req.Interval != "D" || earliest.IsZero() {
		return s.GetBars(ctx, req)
	}
	required := req
	required.From = marketDayOffset(req, earliest, -DefaultDailyMarketDataLookbackDays)
	required.To = marketDayOffset(req, time.Now(), 1)
	if _, ok := s.Provider.(ResponseProvider); ok {
		response, err := s.GetBars(ctx, required)
		response.From, response.To = FormatTimeRFC3339(req.From), FormatTimeRFC3339(req.To)
		return response, err
	}
	key := strings.Join([]string{"daily-coverage-v1", strings.ToUpper(req.Symbol), req.InstrumentType}, "|")

	v, err, _ := s.group.Do(key, func() (any, error) {
		return s.syncCoverage(ctx, required, key)
	})
	if err != nil {
		return Response{}, err
	}
	return responseFor(req, s.Provider.Name(), true, v.([]Bar)), nil
}

func (s *Service) syncCoverage(ctx context.Context, req Request, key string) ([]Bar, error) {
	var bars []Bar
	from, to := req.From, req.To
	cached, err := s.Store.GetMarketBarsCache(ctx, key)
	if err == nil {
		if err := json.Unmarshal(cached.BarsJson, &bars); err != nil {
			bars = nil
		} else if cachedFrom, fromErr := time.Parse(time.RFC3339, cached.FromTs); fromErr == nil {
			if cachedTo, toErr := time.Parse(time.RFC3339, cached.ToTs); toErr == nil {
				from, to = cachedFrom, cachedTo
			}
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	if len(bars) == 0 {
		fetched, err := s.Provider.FetchBars(ctx, req)
		if err != nil {
			return nil, err
		}
		bars = fetched
		from, to = req.From, req.To
	} else {
		if req.From.Before(from) {
			fetched, err := s.Provider.FetchBars(ctx, Request{Symbol: req.Symbol, InstrumentType: req.InstrumentType, Interval: req.Interval, From: req.From, To: from})
			if err != nil {
				return nil, err
			}
			bars = append(bars, fetched...)
			from = req.From
		}
		if req.To.After(to) {
			fetched, err := s.Provider.FetchBars(ctx, Request{Symbol: req.Symbol, InstrumentType: req.InstrumentType, Interval: req.Interval, From: to, To: req.To})
			if err != nil {
				return nil, err
			}
			bars = append(bars, fetched...)
			to = req.To
		}
	}
	bars = dedupeBars(normalizeBars(req, bars))
	raw, err := json.Marshal(bars)
	if err != nil {
		return nil, err
	}
	if err := s.Store.UpsertMarketBarsCache(ctx, store.UpsertMarketBarsCacheParams{
		CacheKey: key, Symbol: req.Symbol, Interval: req.Interval,
		FromTs: FormatTimeRFC3339(from), ToTs: FormatTimeRFC3339(to), BarsJson: raw,
		Provider: s.Provider.Name(), FetchedAt: time.Now().UTC().Format(time.RFC3339),
	}); err != nil {
		slog.Warn("market bars coverage cache write failed", "key", key, "err", err)
	}
	return bars, nil
}

func marketDayOffset(req Request, t time.Time, days int) time.Time {
	loc, err := time.LoadLocation(MarketTimezone(req))
	if err != nil {
		loc = time.UTC
	}
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day()+days, 0, 0, 0, 0, loc).UTC()
}

func dedupeBars(bars []Bar) []Bar {
	byTime := make(map[int64]Bar, len(bars))
	for _, bar := range bars {
		byTime[bar.Time] = bar
	}
	out := make([]Bar, 0, len(byTime))
	for _, bar := range byTime {
		out = append(out, bar)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time < out[j].Time })
	return out
}

func responseFor(req Request, provider string, cached bool, bars []Bar) Response {
	return Response{
		Symbol: req.Symbol, Instrument: req.Symbol, Interval: req.Interval,
		From: FormatTimeRFC3339(req.From), To: FormatTimeRFC3339(req.To),
		Provider: provider, Source: provider, Timezone: MarketTimezone(req),
		AdjustmentStatus: adjustmentStatus(provider), Cached: cached, Bars: normalizeBars(req, bars),
	}
}

func adjustmentStatus(provider string) string {
	return "unadjusted"
}

func (s *Service) fetchBars(ctx context.Context, req Request, key string) (Response, error) {
	if bars, ok := s.mem.get(key); ok {
		return responseFor(req, s.Provider.Name(), true, bars), nil
	}

	cached, err := s.Store.GetMarketBarsCache(ctx, key)
	if err == nil {
		if cached.ExpiresAt.Valid {
			exp, perr := time.Parse(time.RFC3339, cached.ExpiresAt.String)
			if perr == nil && time.Now().UTC().After(exp) {
				goto fetch
			}
		}
		var bars []Bar
		if uerr := json.Unmarshal(cached.BarsJson, &bars); uerr == nil {
			s.mem.set(key, bars, cacheExpiresAt(req.To))
			return responseFor(req, cached.Provider, true, bars), nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return Response{}, err
	}

fetch:
	bars, err := s.Provider.FetchBars(ctx, req)
	if err != nil {
		return Response{}, err
	}
	bars = normalizeBars(req, bars)
	raw, err := json.Marshal(bars)
	if err != nil {
		return Response{}, err
	}
	expires := cacheExpiresAt(req.To)
	var expiresSQL sql.NullString
	if expires != nil {
		expiresSQL = sql.NullString{String: expires.UTC().Format(time.RFC3339), Valid: true}
	}
	if err := s.Store.UpsertMarketBarsCache(ctx, store.UpsertMarketBarsCacheParams{
		CacheKey:  key,
		Symbol:    req.Symbol,
		Interval:  req.Interval,
		FromTs:    FormatTimeRFC3339(req.From),
		ToTs:      FormatTimeRFC3339(req.To),
		BarsJson:  raw,
		Provider:  s.Provider.Name(),
		FetchedAt: time.Now().UTC().Format(time.RFC3339),
		ExpiresAt: expiresSQL,
	}); err != nil {
		// The bars are already in hand — a cache write failure must not
		// turn a good fetch into an error response.
		slog.Warn("market bars cache write failed", "key", key, "err", err)
	}
	s.mem.set(key, bars, expires)
	return responseFor(req, s.Provider.Name(), false, bars), nil
}

func cacheExpiresAt(to time.Time) *time.Time {
	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if to.Before(startOfToday) {
		return nil
	}
	t := now.Add(5 * time.Minute)
	return &t
}

type memEntry struct {
	bars []Bar
	// expires follows the DB cache policy: nil for fully-historical windows
	// (bars older than today never change), otherwise a short TTL so a
	// long-lived process does not serve today's bars stale forever.
	expires *time.Time
}

type memCache struct {
	mu    sync.RWMutex
	max   int
	order []string
	data  map[string]memEntry
}

func newMemCache(max int) *memCache {
	if max < 1 {
		max = 64
	}
	return &memCache{max: max, data: make(map[string]memEntry)}
}

func (c *memCache) get(key string) ([]Bar, bool) {
	c.mu.RLock()
	entry, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if entry.expires != nil && time.Now().UTC().After(*entry.expires) {
		c.mu.Lock()
		if cur, still := c.data[key]; still && cur.expires != nil && time.Now().UTC().After(*cur.expires) {
			delete(c.data, key)
		}
		c.mu.Unlock()
		return nil, false
	}
	out := make([]Bar, len(entry.bars))
	copy(out, entry.bars)
	return out, true
}

func (c *memCache) set(key string, bars []Bar, expires *time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.data[key]; !exists {
		c.order = append(c.order, key)
	}
	cp := make([]Bar, len(bars))
	copy(cp, bars)
	c.data[key] = memEntry{bars: cp, expires: expires}
	for len(c.order) > c.max {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.data, oldest)
	}
}

// NewProvider picks a market data provider from config.
func NewProvider(providerName, apiKey string, httpConfig ...string) Provider {
	switch providerName {
	case "http":
		var baseURL, key string
		if len(httpConfig) > 0 {
			baseURL = httpConfig[0]
		}
		if len(httpConfig) > 1 {
			key = httpConfig[1]
		}
		return NewHTTPProvider(baseURL, key)
	case "finnhub":
		if apiKey != "" {
			return NewFinnhubProvider(apiKey)
		}
		return NewYahooProvider()
	case "yahoo", "":
		return NewYahooProvider()
	default:
		if apiKey != "" {
			return NewFinnhubProvider(apiKey)
		}
		return NewYahooProvider()
	}
}
