package api

import (
	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/coach"
	"net/http"
)

// Authenticated capability never contains instance-owner configuration.
func (s *Server) handleCoachAvailability(c *echo.Context) error {
	c.Response().Header().Set("Cache-Control", "no-store")
	return c.JSON(http.StatusOK, map[string]any{
		"available": coach.LegacyAvailable(), "reason": "unavailable",
		"news_analysis_available": false,
	})
}
