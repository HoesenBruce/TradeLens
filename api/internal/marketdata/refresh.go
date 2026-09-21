package marketdata

import (
	"context"
	"errors"
	"time"
)

// RefreshBars bypasses cached windows for evaluations requiring post-close evidence.
// The source acquisition time is distinct from when a local HTTP archive is read.
func (s *Service) RefreshBars(ctx context.Context, req Request) (Response, error) {
	if s == nil || s.Provider == nil {
		return Response{}, errors.New("market data provider not configured")
	}
	if p, ok := s.Provider.(ResponseProvider); ok {
		return p.FetchResponse(ctx, req)
	}
	started := time.Now().UTC()
	bars, err := s.Provider.FetchBars(ctx, req)
	if err != nil {
		return Response{}, err
	}
	response := responseFor(req, s.Provider.Name(), false, bars)
	response.Instrument = chartSymbol(req)
	response.FetchedAt = &started
	return response, nil
}
