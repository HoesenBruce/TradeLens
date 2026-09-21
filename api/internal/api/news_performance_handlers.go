package api

import (
	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"net/http"
	"strconv"
)

func (s *Server) handleNewsPerformance(c *echo.Context) error {
	f := predictionvalidation.PerformanceFilter{Source: c.QueryParam("source"), Symbol: c.QueryParam("symbol"), AssetType: c.QueryParam("asset_type"), Category: c.QueryParam("category"), From: c.QueryParam("from"), To: c.QueryParam("to")}
	if h := c.QueryParam("horizon"); h != "" {
		var err error
		f.Horizon, err = strconv.ParseInt(h, 10, 64)
		if err != nil || f.Horizon == 0 {
			return Fail(400, "bad_request", "invalid horizon", nil)
		}
	}
	if err := f.Validate(); err != nil {
		return Fail(400, "bad_request", err.Error(), nil)
	}
	engine := predictionvalidation.Engine{Store: s.deps.Store}
	result, err := engine.Performance(c.Request().Context(), auth.UserID(c), f)
	if err != nil {
		return Fail(500, "internal", "could not load prediction performance", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, result)
}
