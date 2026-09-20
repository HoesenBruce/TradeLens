package api

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strings"
	"time"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/store"
)

type predictionBody struct {
	NewsAssetID  string  `json:"news_asset_id"`
	Source       string  `json:"source"`
	Direction    string  `json:"direction"`
	Confidence   *int64  `json:"confidence"`
	Reasoning    string  `json:"reasoning"`
	Catalysts    string  `json:"catalysts"`
	Risks        string  `json:"risks"`
	Invalidation string  `json:"invalidation"`
	Horizons     []int64 `json:"horizons"`
}

type predictionDTO struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	predictionBody
}

func normalizePrediction(in predictionBody) (predictionBody, *Error) {
	if in.Source != "" && in.Source != "user" {
		return in, Fail(http.StatusBadRequest, "bad_request", "manual predictions must have user source", nil)
	}
	in.Source = "user"
	if in.Direction != "bullish" && in.Direction != "bearish" && in.Direction != "neutral" {
		return in, Fail(http.StatusBadRequest, "bad_request", "direction must be bullish, bearish, or neutral", nil)
	}
	if in.Confidence != nil && (*in.Confidence < 0 || *in.Confidence > 100) {
		return in, Fail(http.StatusBadRequest, "bad_request", "confidence must be an integer from 0 to 100", nil)
	}
	if len(in.Horizons) == 0 {
		return in, Fail(http.StatusBadRequest, "bad_request", "select at least one trading-day horizon", nil)
	}
	seen := map[int64]bool{}
	for _, h := range in.Horizons {
		if (h != 1 && h != 3 && h != 5 && h != 10 && h != 20) || seen[h] {
			return in, Fail(http.StatusBadRequest, "bad_request", "horizons must be unique values from 1, 3, 5, 10, 20", nil)
		}
		seen[h] = true
	}
	in.Reasoning = strings.TrimSpace(in.Reasoning)
	in.Catalysts = strings.TrimSpace(in.Catalysts)
	in.Risks = strings.TrimSpace(in.Risks)
	in.Invalidation = strings.TrimSpace(in.Invalidation)
	return in, nil
}

func predictionConfidence(in *int64) sql.NullInt64 {
	if in == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *in, Valid: true}
}

func loadPredictionDTO(ctx context.Context, q store.Querier, p store.Prediction, userID string) (predictionDTO, error) {
	out := predictionDTO{ID: p.ID, CreatedAt: p.CreatedAt, UpdatedAt: p.UpdatedAt, predictionBody: predictionBody{NewsAssetID: p.NewsAssetID, Source: p.Source, Direction: p.Direction, Reasoning: p.Reasoning, Catalysts: p.Catalysts, Risks: p.Risks, Invalidation: p.Invalidation, Horizons: []int64{}}}
	if p.Confidence.Valid {
		out.Confidence = &p.Confidence.Int64
	}
	horizons, err := q.ListPredictionHorizons(ctx, store.ListPredictionHorizonsParams{PredictionID: p.ID, UserID: userID})
	if err != nil {
		return out, err
	}
	for _, h := range horizons {
		out.Horizons = append(out.Horizons, h.TradingDays)
	}
	return out, nil
}

func loadPredictions(ctx context.Context, q store.Querier, newsID, userID string) ([]predictionDTO, error) {
	rows, err := q.ListPredictions(ctx, store.ListPredictionsParams{NewsID: newsID, UserID: userID})
	if err != nil {
		return nil, err
	}
	out := make([]predictionDTO, 0, len(rows))
	for _, p := range rows {
		dto, err := loadPredictionDTO(ctx, q, p, userID)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, nil
}

func (s *Server) handleSavePrediction(c *echo.Context) error {
	var in predictionBody
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid body", nil)
	}
	in, apiErr := normalizePrediction(in)
	if apiErr != nil {
		return apiErr
	}
	ctx, userID, newsID := c.Request().Context(), auth.UserID(c), c.Param("id")
	id := c.Param("predictionId")
	creating := id == ""
	if creating {
		id = uuid.New().String()
	}
	var out predictionDTO
	err := store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		if _, err := q.GetNews(ctx, store.GetNewsParams{ID: newsID, UserID: userID}); err != nil {
			return err
		}
		var p store.Prediction
		var err error
		if creating {
			a, err := q.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: in.NewsAssetID, UserID: userID})
			if err != nil {
				return err
			}
			if a.NewsID != newsID {
				return sql.ErrNoRows
			}
			p, err = q.CreatePrediction(ctx, store.CreatePredictionParams{ID: id, NewsAssetID: a.ID, Source: "user", Direction: in.Direction, Confidence: predictionConfidence(in.Confidence), Reasoning: in.Reasoning, Catalysts: in.Catalysts, Risks: in.Risks, Invalidation: in.Invalidation, UserID: userID})
		} else {
			previous, err := q.GetPrediction(ctx, store.GetPredictionParams{ID: id, UserID: userID})
			if err != nil {
				return err
			}
			a, err := q.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: previous.NewsAssetID, UserID: userID})
			if err != nil {
				return err
			}
			if a.NewsID != newsID || previous.Source != "user" {
				return sql.ErrNoRows
			}
			if in.NewsAssetID != "" && in.NewsAssetID != previous.NewsAssetID {
				return Fail(http.StatusBadRequest, "bad_request", "prediction asset cannot change", nil)
			}
			p, err = q.UpdatePrediction(ctx, store.UpdatePredictionParams{ID: id, UserID: userID, Source: "user", Direction: in.Direction, Confidence: predictionConfidence(in.Confidence), Reasoning: in.Reasoning, Catalysts: in.Catalysts, Risks: in.Risks, Invalidation: in.Invalidation})
		}
		if err != nil {
			return err
		}
		if err = q.DeletePredictionHorizons(ctx, store.DeletePredictionHorizonsParams{PredictionID: id, UserID: userID}); err != nil {
			return err
		}
		for _, h := range in.Horizons {
			if err = q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: id, TradingDays: h, UserID: userID}); err != nil {
				return err
			}
		}
		out, err = loadPredictionDTO(ctx, q, p, userID)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news, asset, or user prediction not found", nil)
	}
	var apiError *Error
	if errors.As(err, &apiError) {
		return apiError
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not save prediction", nil).Wrap(err)
	}
	status := http.StatusOK
	if creating {
		status = http.StatusCreated
	}
	return c.JSON(status, out)
}

func (s *Server) handleDeletePrediction(c *echo.Context) error {
	ctx, userID := c.Request().Context(), auth.UserID(c)
	err := store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		p, err := q.GetPrediction(ctx, store.GetPredictionParams{ID: c.Param("predictionId"), UserID: userID})
		if err != nil {
			return err
		}
		a, err := q.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: p.NewsAssetID, UserID: userID})
		if err != nil {
			return err
		}
		if a.NewsID != c.Param("id") || p.Source != "user" {
			return sql.ErrNoRows
		}
		n, err := q.DeletePrediction(ctx, store.DeletePredictionParams{ID: p.ID, UserID: userID, Source: "user"})
		if err != nil {
			return err
		}
		if n == 0 {
			return sql.ErrNoRows
		}
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "user prediction not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not delete prediction", nil).Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}
