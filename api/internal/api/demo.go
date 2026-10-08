package api

import (
	"net/http"

	"github.com/labstack/echo/v5"
)

// Fail closed: new routes need an explicit demo review, even if they use GET.
func demoReadOnly(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c *echo.Context) error {
		path := c.Path()
		allowed := false
		if c.Request().Method == http.MethodPost {
			allowed = path == "/api/v1/auth/login" || path == "/api/v1/auth/refresh"
		}
		if c.Request().Method == http.MethodGet {
			switch path {
			case "/healthz", "/api/v1/setup/status", "/api/v1/me", "/api/v1/me/preferences",
				"/api/v1/accounts", "/api/v1/accounts/:id", "/api/v1/cash-transactions",
				"/api/v1/executions", "/api/v1/trades", "/api/v1/trades/:id",
				"/api/v1/tags", "/api/v1/setups", "/api/v1/setups/:id",
				"/api/v1/notes", "/api/v1/notes/:id", "/api/v1/news", "/api/v1/news/:id",
				"/api/v1/news/performance", "/api/v1/news/:id/predictions/:predictionId/validations",
				"/api/v1/market/bars", "/api/v1/market/corporate-actions",
				"/api/v1/analytics/summary", "/api/v1/analytics/r-summary", "/api/v1/analytics/equity-curve",
				"/api/v1/analytics/account-value", "/api/v1/analytics/daily", "/api/v1/analytics/breakdown",
				"/api/v1/analytics/compliance", "/api/v1/analytics/behavior",
				"/api/v1/analytics/montecarlo", "/api/v1/analytics/execution-score",
				"/api/v1/settings/risk-rules", "/api/v1/settings/annual-goal", "/api/v1/settings/checklist-template",
				"/api/v1/system/info":
				allowed = true
			}
		}
		if !allowed {
			return Fail(http.StatusForbidden, "demo_read_only", "Public demo is read-only; uploads, credentials and administration are disabled", nil)
		}
		return next(c)
	}
}
