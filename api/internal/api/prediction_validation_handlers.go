package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/predictionvalidation"
	"io"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handlePredictionValidation(c *echo.Context) error {
	engine := predictionvalidation.Engine{Market: s.deps.Market, Store: s.deps.Store}
	ctx, owner, news, prediction := c.Request().Context(), auth.UserID(c), c.Param("id"), c.Param("predictionId")
	var result []predictionvalidation.Evaluation
	var err error
	if c.Request().Method == http.MethodPost {
		var body struct {
			Benchmark *predictionvalidation.Benchmark `json:"benchmark"`
		}
		decoder := json.NewDecoder(http.MaxBytesReader(c.Response(), c.Request().Body, 4096))
		decoder.DisallowUnknownFields()
		if decodeErr := decoder.Decode(&body); decodeErr != nil && !errors.Is(decodeErr, io.EOF) {
			return Fail(http.StatusBadRequest, "bad_request", "invalid benchmark body", nil)
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			return Fail(http.StatusBadRequest, "bad_request", "invalid benchmark body", nil)
		}
		if body.Benchmark != nil && (strings.TrimSpace(body.Benchmark.Symbol) == "" || strings.TrimSpace(body.Benchmark.Market) == "" || strings.TrimSpace(body.Benchmark.Currency) == "") {
			return Fail(http.StatusBadRequest, "bad_request", "benchmark requires symbol, market and currency", nil)
		}
		result, err = engine.Validate(ctx, owner, news, prediction, time.Now().UTC(), body.Benchmark)
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
