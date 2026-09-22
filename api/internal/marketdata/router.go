package marketdata

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// ErrProviderUnavailable marks an explicit temporary upstream outage or rate limit.
var ErrProviderUnavailable = errors.New("market data provider unavailable")

func unavailableStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable || status == http.StatusGatewayTimeout
}

// Router tries whole responses in order; it never merges bars from different sources.
// ponytail: fresh responses preserve provenance but increase upstream requests;
// add a full-response cache if routing traffic requires it.
type Router struct {
	providers []Provider
}

// NewConfiguredProvider opts into routing only when an explicit CSV order is set.
// The legacy single-provider factory and its defaults remain unchanged.
func NewConfiguredProvider(name, order, apiKey, baseURL, httpKey string) (Provider, error) {
	if strings.TrimSpace(order) == "" {
		return NewProvider(name, apiKey, baseURL, httpKey), nil
	}
	router := &Router{}
	seen := map[string]bool{}
	for _, name := range strings.Split(order, ",") {
		name = strings.TrimSpace(name)
		if seen[name] {
			return nil, fmt.Errorf("duplicate market data provider %q", name)
		}
		seen[name] = true
		switch name {
		case "yahoo":
		case "finnhub":
			if strings.TrimSpace(apiKey) == "" {
				return nil, errors.New("finnhub routing requires a market data API key")
			}
		case "http":
			u, err := url.Parse(baseURL)
			if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
				return nil, errors.New("http routing requires a valid market data HTTP base URL")
			}
		default:
			return nil, fmt.Errorf("unknown market data provider %q", name)
		}
		router.providers = append(router.providers, NewProvider(name, apiKey, baseURL, httpKey))
	}
	return router, nil
}

func (r *Router) Name() string { return "router" }

func (r *Router) FetchBars(ctx context.Context, req Request) ([]Bar, error) {
	response, err := r.FetchResponse(ctx, req)
	return response.Bars, err
}

func (r *Router) FetchResponse(ctx context.Context, req Request) (Response, error) {
	var empty Response
	var failures []error
	for _, provider := range r.providers {
		if err := ctx.Err(); err != nil {
			return Response{}, err
		}
		// Reuse the shared fresh-fetch path, including legacy acquisition metadata.
		response, err := (&Service{Provider: provider}).RefreshBars(ctx, req)
		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		if err != nil {
			if !errors.Is(err, ErrProviderUnavailable) && !errors.Is(err, ErrUnsupportedResolution) {
				return Response{}, err
			}
			failures = append(failures, fmt.Errorf("%s: %w", provider.Name(), err))
			continue
		}
		response.Provider = provider.Name()
		if len(response.Bars) > 0 {
			return response, nil
		}
		if empty.Provider == "" {
			empty = response
		}
	}
	// An outage must not become a successful empty window just because another
	// provider had no data. Only an all-empty chain is a successful no-data result.
	if len(failures) > 0 {
		return Response{}, errors.Join(failures...)
	}
	if empty.Provider == "" {
		return Response{}, errors.New("market data router has no providers")
	}
	return empty, nil
}
