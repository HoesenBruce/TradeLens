package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
)

func TestDemoPolicy(t *testing.T) {
	for _, tc := range []struct {
		method, path string
		allowed      bool
	}{
		{"POST", "/api/v1/auth/login", true},
		{"POST", "/api/v1/auth/refresh", true},
		{"GET", "/api/v1/trades/:id", true},
		{"GET", "/api/v1/news/:id/predictions/:predictionId/validations", true},
		{"GET", "/api/v1/analytics/account-value", true},
		{"POST", "/api/v1/setup", false},
		{"POST", "/api/v1/auth/register", false},
		{"PUT", "/api/v1/me/password", false},
		{"POST", "/api/v1/me/totp/start", false},
		{"GET", "/api/v1/access-tokens", false},
		{"POST", "/api/v1/imports/commit", false},
		{"POST", "/api/v1/trades/:id/attachments", false},
		{"POST", "/api/v1/news/:id/analyze", false},
		{"DELETE", "/api/v1/accounts/:id", false},
		{"GET", "/api/v1/admin/users", false},
		{"GET", "/api/v1/settings/coach", false},
		{"GET", "/api/v1/accounts/:id/flex-sync", false},
		{"GET", "/api/v1/future-route", false},
		{"HEAD", "/api/v1/trades", false},
	} {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			e := echo.New()
			c := e.NewContext(httptest.NewRequest(tc.method, "/", nil), httptest.NewRecorder())
			c.SetPath(tc.path)
			called := false
			err := demoReadOnly(func(c *echo.Context) error { called = true; return nil })(c)
			if called != tc.allowed || (err == nil) != tc.allowed {
				t.Fatalf("allowed=%v called=%v err=%v", tc.allowed, called, err)
			}
		})
	}
}

func TestDemoKeepsAuthentication(t *testing.T) {
	s := New(Deps{DemoMode: true, JWT: auth.NewJWT("test")})
	// Allowed reads still pass through the normal JWT middleware.
	rec := httptest.NewRecorder()
	s.Echo.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/accounts", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("accounts: %d", rec.Code)
	}
}
