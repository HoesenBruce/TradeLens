package marketdata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type routeProvider struct {
	name     string
	response Response
	err      error
	calls    int
}

func (p *routeProvider) Name() string { return p.name }
func (p *routeProvider) FetchBars(ctx context.Context, req Request) ([]Bar, error) {
	response, err := p.FetchResponse(ctx, req)
	return response.Bars, err
}
func (p *routeProvider) FetchResponse(context.Context, Request) (Response, error) {
	p.calls++
	return p.response, p.err
}

func TestRouterFallbackPolicy(t *testing.T) {
	fatal := errors.New("invalid data")
	for _, tc := range []struct {
		name      string
		firstErr  error
		firstData bool
		fallback  bool
		wantErr   error
	}{
		{name: "first wins", firstData: true},
		{name: "empty", fallback: true},
		{name: "unavailable", firstErr: fmt.Errorf("wrapped: %w", ErrProviderUnavailable), fallback: true},
		{name: "unsupported", firstErr: ErrUnsupportedResolution, fallback: true},
		{name: "quality", firstErr: fatal, wantErr: fatal},
		{name: "canceled", firstErr: context.Canceled, wantErr: context.Canceled},
		{name: "deadline", firstErr: context.DeadlineExceeded, wantErr: context.DeadlineExceeded},
	} {
		t.Run(tc.name, func(t *testing.T) {
			first := &routeProvider{name: "first", err: tc.firstErr}
			if tc.firstData {
				first.response.Bars = []Bar{{Close: 100}}
			}
			stamp := time.Now().UTC()
			second := &routeProvider{name: "second", response: Response{Source: "archive", Timezone: "Asia/Tokyo", AdjustmentStatus: "adjusted", FetchedAt: &stamp, Bars: []Bar{{Close: 123, SplitRatio: 2}}}}
			router := &Router{providers: []Provider{first, second}}
			got, err := router.FetchResponse(context.Background(), httpRequest())
			require.ErrorIs(t, err, tc.wantErr)
			if tc.fallback {
				require.Equal(t, 1, second.calls)
				want := second.response
				want.Provider = "second"
				require.Equal(t, want, got)
			} else {
				require.Zero(t, second.calls)
				if tc.wantErr == nil {
					require.Equal(t, "first", got.Provider)
				}
			}
		})
	}
}

func TestRouterExhaustionAndCancellation(t *testing.T) {
	for _, failure := range []error{nil, ErrProviderUnavailable, ErrUnsupportedResolution} {
		first := &routeProvider{name: "first", err: failure, response: Response{Source: "empty-source", Bars: []Bar{}}}
		second := &routeProvider{name: "second", response: Response{Bars: []Bar{}}}
		router := &Router{providers: []Provider{first, second}}
		got, err := router.FetchResponse(context.Background(), httpRequest())
		require.ErrorIs(t, err, failure)
		if failure == nil {
			require.Equal(t, "first", got.Provider)
			require.Equal(t, "empty-source", got.Source)
			require.Empty(t, got.Bars)
		}
		require.Equal(t, 1, second.calls)
	}
	p := &routeProvider{name: "unused"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := (&Router{providers: []Provider{p}}).FetchBars(ctx, httpRequest())
	require.ErrorIs(t, err, context.Canceled)
	require.Zero(t, p.calls)
	_, err = (&Router{}).FetchBars(context.Background(), httpRequest())
	require.ErrorContains(t, err, "no providers")
}

func TestConfiguredProvider(t *testing.T) {
	for _, name := range []string{"", "yahoo", "finnhub", "http", "legacy-unknown"} {
		p, err := NewConfiguredProvider(name, "", "", "http://localhost", "")
		require.NoError(t, err)
		require.IsType(t, NewProvider(name, "", "http://localhost", ""), p)
	}
	p, err := NewConfiguredProvider("yahoo", " http, finnhub, yahoo ", "key", "https://example.com/archive", "secret")
	require.NoError(t, err)
	router := p.(*Router)
	require.Equal(t, []string{"http", "finnhub", "yahoo"}, []string{router.providers[0].Name(), router.providers[1].Name(), router.providers[2].Name()})
	require.Equal(t, "secret", router.providers[0].(*HTTPProvider).APIKey)
	for _, tc := range []struct{ order, key, url string }{
		{order: "typo"}, {order: "yahoo,"}, {order: "yahoo,yahoo"}, {order: "finnhub"},
		{order: "http"}, {order: "http", url: "https://user:secret@example.com"},
	} {
		_, err := NewConfiguredProvider("yahoo", tc.order, tc.key, tc.url, "")
		require.Error(t, err)
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestRouterHTTPThroughService(t *testing.T) {
	for _, tc := range []struct {
		name     string
		status   int
		body     string
		fallback bool
	}{
		{"empty", 200, `{"symbol":"AAPL","interval":"D","bars":[]}`, true},
		{"rate limit", 429, `secret`, true}, {"bad gateway", 502, `secret`, true},
		{"unavailable", 503, `secret`, true}, {"gateway timeout", 504, `secret`, true},
		{"unsupported", 422, `{"error":{"code":"unsupported_interval"}}`, true},
		{"auth", 401, `secret`, false}, {"forbidden", 403, `secret`, false},
		{"invalid request", 400, `secret`, false}, {"unknown server error", 500, `secret`, false},
		{"unknown 422", 422, `secret`, false}, {"missing bars", 200, `{}`, false},
		{"partial bar", 200, `{"symbol":"AAPL","interval":"D","bars":[{"close":100}]}`, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(tc.status); fmt.Fprint(w, tc.body) }))
			defer server.Close()
			last := &recordingProvider{}
			svc := NewService(nil, &Router{providers: []Provider{NewHTTPProvider(server.URL, ""), last}})
			got, err := svc.GetBars(context.Background(), httpRequest())
			if tc.fallback {
				require.NoError(t, err)
				require.Equal(t, last.Name(), got.Provider)
				require.Equal(t, last.Name(), got.Source)
				require.NotNil(t, got.FetchedAt)
				require.Len(t, last.requests, 1)
			} else {
				require.Error(t, err)
				require.NotContains(t, err.Error(), "secret")
				require.Empty(t, last.requests)
			}
		})
	}
	// Metadata must survive both transaction coverage and fresh validation paths.
	stamp := time.Now().UTC()
	p := &routeProvider{name: "http", response: Response{Source: "archive", FetchedAt: &stamp, AdjustmentStatus: "unknown", Bars: []Bar{{Close: 100}}}}
	svc := NewService(nil, &Router{providers: []Provider{p}})
	req := httpRequest()
	for _, fetch := range []func() (Response, error){
		func() (Response, error) { return svc.GetTransactionBars(context.Background(), req, req.From) },
		func() (Response, error) { return svc.RefreshBars(context.Background(), req) },
	} {
		got, err := fetch()
		require.NoError(t, err)
		require.Equal(t, "http", got.Provider)
		require.Equal(t, "archive", got.Source)
		require.Equal(t, &stamp, got.FetchedAt)
	}
}

type routeTransport func(*http.Request) (*http.Response, error)

func (f routeTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestLegacyProviderFallbackClassification(t *testing.T) {
	for _, name := range []string{"yahoo", "finnhub"} {
		for _, tc := range []struct {
			name           string
			status         int
			yahoo, finnhub string
			fallback       bool
		}{
			{"empty", 200, `{"chart":{"result":[]}}`, `{"s":"no_data"}`, true},
			{"unavailable", 503, `secret`, `secret`, true},
			{"rate limit", 429, `secret`, `secret`, true},
			{"auth", 401, `denied`, `denied`, false},
			{"invalid range", 422, `invalid range`, `invalid range`, false},
			{"missing timestamps", 200, `{"chart":{"result":[{}]}}`, `{"s":"ok"}`, false},
			{"malformed", 200, `{`, `{`, false},
			{"missing", 200, `{}`, `{}`, false},
			{"unknown error", 200, `{"chart":{"error":{"code":"Unauthorized"}}}`, `{"s":"error"}`, false},
			{"partial", 200, `{"chart":{"result":[{"timestamp":[1],"indicators":{"quote":[{"open":[],"high":[2],"low":[1],"close":[1]}]}}]}}`, `{"s":"ok","t":[1],"o":[],"h":[2],"l":[1],"c":[1],"v":[1]}`, false},
		} {
			t.Run(name+"/"+tc.name, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tc.status)
					if name == "yahoo" {
						fmt.Fprint(w, tc.yahoo)
					} else {
						fmt.Fprint(w, tc.finnhub)
					}
				}))
				defer server.Close()
				var p Provider
				if name == "yahoo" {
					p = &YahooProvider{Client: server.Client(), ChartBase: server.URL}
				} else {
					client := &http.Client{Transport: routeTransport(func(r *http.Request) (*http.Response, error) {
						local, err := http.NewRequestWithContext(r.Context(), r.Method, server.URL, nil)
						if err != nil {
							return nil, err
						}
						return server.Client().Do(local)
					})}
					p = &FinnhubProvider{APIKey: "key", Client: client}
				}
				last := &recordingProvider{}
				_, err := (&Router{providers: []Provider{p, last}}).FetchBars(context.Background(), httpRequest())
				if tc.fallback {
					require.NoError(t, err)
					require.Len(t, last.requests, 1)
				} else {
					require.Error(t, err)
					require.Empty(t, last.requests)
				}
			})
		}
	}
}
