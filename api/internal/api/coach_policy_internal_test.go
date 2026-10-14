package api

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/tradermemos/api/internal/coach"
	"github.com/tradermemos/api/internal/ocr"
	"testing"
)

func TestLegacyPersistenceCannotBypassAvailability(t *testing.T) {
	// No store is needed: the policy must refuse before marshaling or writing.
	s := &Server{}
	_, err := s.saveCoachReview(context.Background(), "owner", "trade", ocr.VisionConfig{Enabled: true}, coach.Review{NextAction: "synthetic-private"})
	require.ErrorIs(t, err, coach.ErrUnavailable)
}
