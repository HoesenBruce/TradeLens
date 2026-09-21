package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/coach"
	"github.com/tradermemos/api/internal/store"
)

func (s *Server) handleAnalyzeNews(c *echo.Context) error {
	ctx, user := c.Request().Context(), auth.UserID(c)
	n, err := s.deps.Store.GetNews(ctx, store.GetNewsParams{ID: c.Param("id"), UserID: user})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(404, "not_found", "news not found", nil)
	}
	if err != nil {
		return Fail(500, "internal", "could not load news", nil).Wrap(err)
	}
	input, _ := json.Marshal(map[string]any{"title": n.Title, "source": n.Source, "published_at": n.PublishedAt, "original_text": n.OriginalText, "notes": n.Notes})
	result, err := coach.AnalyzeNews(ctx, s.effectiveCoachConfig(ctx), string(input))
	if errors.Is(err, coach.ErrUnavailable) {
		return Fail(503, "ai_unavailable", "Configure and enable AI Coach in Settings first.", nil)
	}
	if errors.Is(err, coach.ErrTimeout) {
		return Fail(504, "ai_timeout", "AI analysis timed out. No data was changed.", nil)
	}
	if err != nil {
		return Fail(502, "ai_failed", "AI analysis failed or returned invalid suggestions. No data was changed.", nil)
	}
	return c.JSON(http.StatusOK, result)
}

type reviewedNewsAsset struct {
	coach.NewsAssetSuggestion
	Source            string `json:"source"`
	IncludePrediction bool   `json:"include_prediction"`
}
type acceptNewsAnalysisBody struct {
	Summary          *string             `json:"summary"`
	Category         *string             `json:"category"`
	ExpectedSummary  string              `json:"expected_summary"`
	ExpectedCategory string              `json:"expected_category"`
	Assets           []reviewedNewsAsset `json:"assets"`
}

func (s *Server) handleAcceptNewsAnalysis(c *echo.Context) error {
	var in acceptNewsAnalysisBody
	if err := c.Bind(&in); err != nil {
		return Fail(400, "bad_request", "invalid review", nil)
	}
	if len(in.Assets) > 20 {
		return Fail(400, "bad_request", "at most 20 assets per review", nil)
	}
	for _, a := range in.Assets {
		if a.Source != "user" && a.Source != "ai" {
			return Fail(400, "bad_request", "review source must be user or ai", nil)
		}
		if _, err := normalizeNewsAsset(newsAssetBody{AssetType: a.AssetType, Symbol: a.Symbol, Source: a.Source}); err != nil {
			return err
		}
		if a.IncludePrediction {
			if err := (coach.NewsAnalysis{Summary: "review", Category: "review", Assets: []coach.NewsAssetSuggestion{a.NewsAssetSuggestion}}).Validate(); err != nil {
				return Fail(400, "bad_request", err.Error(), nil)
			}
		}
	}
	ctx, user, id := c.Request().Context(), auth.UserID(c), c.Param("id")
	var result newsDTO
	err := store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		n, err := q.GetNews(ctx, store.GetNewsParams{ID: id, UserID: user})
		if err != nil {
			return err
		}
		// Compare selected fields with the review's baseline before replacing them.
		if (in.Summary != nil && n.Summary != in.ExpectedSummary) || (in.Category != nil && n.Category != in.ExpectedCategory) {
			return Fail(409, "conflict", "News changed during review. Reopen the review before saving.", nil)
		}
		if in.Summary != nil {
			n.Summary = strings.TrimSpace(*in.Summary)
		}
		if in.Category != nil {
			n.Category = strings.TrimSpace(*in.Category)
		}
		if in.Summary != nil || in.Category != nil {
			_, err = q.UpdateNews(ctx, store.UpdateNewsParams{ID: id, UserID: user, Title: n.Title, Source: n.Source, Url: n.Url, PublishedAt: n.PublishedAt, OriginalText: n.OriginalText, Notes: n.Notes, Summary: n.Summary, Category: n.Category, Tags: n.Tags})
			if err != nil {
				return err
			}
		}
		for _, a := range in.Assets {
			body, _ := normalizeNewsAsset(newsAssetBody{AssetType: a.AssetType, Symbol: a.Symbol, Market: a.Market, Exchange: a.Exchange, DisplayName: a.DisplayName, Source: a.Source})
			asset, err := q.CreateNewsAsset(ctx, assetParams(uuid.New().String(), id, user, body))
			if err != nil {
				return err
			}
			if a.IncludePrediction {
				if err := createReviewedPrediction(ctx, q, user, asset.ID, a); err != nil {
					return err
				}
			}
		}
		result, err = s.loadNewsDTO(ctx, q, id, user)
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(404, "not_found", "news not found", nil)
	}
	var apiErr *Error
	if errors.As(err, &apiErr) {
		return apiErr
	}
	if err != nil {
		return Fail(500, "internal", "could not save review", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, result)
}

func createReviewedPrediction(ctx context.Context, q store.Querier, user, asset string, a reviewedNewsAsset) error {
	id := uuid.New().String()
	_, err := q.CreatePrediction(ctx, store.CreatePredictionParams{ID: id, NewsAssetID: asset, UserID: user, Source: a.Source, Direction: a.Direction, Confidence: predictionConfidence(a.Confidence), Reasoning: strings.TrimSpace(a.Reasoning), Catalysts: strings.TrimSpace(a.Catalysts), Risks: strings.TrimSpace(a.Risks)})
	if err != nil {
		return err
	}
	for _, h := range a.Horizons {
		if err := q.AddPredictionHorizon(ctx, store.AddPredictionHorizonParams{PredictionID: id, TradingDays: h, UserID: user}); err != nil {
			return err
		}
	}
	return nil
}
