package providererrors

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafePreservesIdentityWithoutDiagnostics(t *testing.T) {
	secret := errors.New("Bearer synthetic-key; private-prompt; webhook-secret; ibkr-token")
	for _, original := range []error{secret, &url.Error{Op: "Get", URL: "https://private.invalid/?t=ibkr-token", Err: secret}, fmt.Errorf("%w: %w", secret, context.Canceled), fmt.Errorf("%w: %w", secret, context.DeadlineExceeded), fmt.Errorf("%w: %w", secret, HTTPStatus(401))} {
		safe := Safe(original)
		require.ErrorIs(t, safe, secret)
		var log bytes.Buffer
		slog.New(slog.NewJSONHandler(&log, nil)).Warn("provider failed", "err", safe)
		for _, output := range []string{fmt.Sprintf("%v %+v", safe, safe), log.String()} {
			for _, value := range []string{"synthetic-key", "private-prompt", "webhook-secret", "ibkr-token", "private.invalid"} {
				require.False(t, bytes.Contains([]byte(output), []byte(value)), "formatted diagnostic leaked a sentinel")
			}
		}
	}
	require.ErrorIs(t, Safe(context.Canceled), context.Canceled)
	require.ErrorIs(t, Safe(context.DeadlineExceeded), context.DeadlineExceeded)
	require.EqualError(t, Safe(HTTPStatus(401)), "external service returned HTTP 401")
	require.Nil(t, Safe(nil))
}
