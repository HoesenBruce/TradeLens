package api

import (
	"database/sql"
	"errors"
	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"net/http"
	"time"
)

func (s *Server) handlePredictionValidation(c *echo.Context) error {
	engine := predictionvalidation.Engine{Market: s.deps.Market, Store: s.deps.Store}
	ctx, owner, news, prediction := c.Request().Context(), auth.UserID(c), c.Param("id"), c.Param("predictionId")
	var result []predictionvalidation.Evaluation
	var err error
	if c.Request().Method == http.MethodPost {
		result, err = engine.Validate(ctx, owner, news, prediction, time.Now().UTC())
	} else {
		result, err = engine.History(ctx, owner, news, prediction)
	}
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "prediction not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not evaluate prediction", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, result)
}
