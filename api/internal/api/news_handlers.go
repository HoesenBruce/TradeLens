package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
	"uuid"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/store"
)

func (s *Server) newsRoutes(g *echo.Group) {
	g.POST("/news/:id/predictions", s.handleSavePrediction)
	g.POST("/news/:id/predictions/:predictionId/validate", s.handlePredictionValidation)
	g.GET("/news/:id/predictions/:predictionId/validations", s.handlePredictionValidation)
	g.PATCH("/news/:id/predictions/:predictionId", s.handleSavePrediction)
	g.DELETE("/news/:id/predictions/:predictionId", s.handleDeletePrediction)
	g.POST("/news", s.handleCreateNews)
	g.GET("/news", s.handleListNews)
	g.GET("/news/:id", s.handleGetNews)
	g.PATCH("/news/:id", s.handleUpdateNews)
	g.DELETE("/news/:id", s.handleDeleteNews)
	g.POST("/news/:id/assets", s.handleCreateNewsAsset)
	g.PATCH("/news/:id/assets/:assetId", s.handleUpdateNewsAsset)
	g.DELETE("/news/:id/assets/:assetId", s.handleDeleteNewsAsset)
}

type newsAssetBody struct {
	AssetType   string `json:"asset_type"`
	Symbol      string `json:"symbol"`
	Market      string `json:"market"`
	Exchange    string `json:"exchange"`
	DisplayName string `json:"display_name"`
	Relation    string `json:"relation"`
	Source      string `json:"source"`
}

type newsAssetDTO struct {
	ID        string    `json:"id"`
	NewsID    string    `json:"news_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	newsAssetBody
}

type newsBody struct {
	Title        string          `json:"title"`
	Source       string          `json:"source"`
	URL          string          `json:"url"`
	PublishedAt  time.Time       `json:"published_at"`
	OriginalText string          `json:"original_text"`
	Notes        string          `json:"notes"`
	Summary      string          `json:"summary"`
	Category     string          `json:"category"`
	Tags         []string        `json:"tags"`
	Assets       []newsAssetBody `json:"assets"`
}

type newsDTO struct {
	Predictions  []predictionDTO `json:"predictions"`
	ID           string          `json:"id"`
	UserID       string          `json:"user_id"`
	Title        string          `json:"title"`
	Source       string          `json:"source"`
	URL          string          `json:"url"`
	PublishedAt  time.Time       `json:"published_at"`
	OriginalText string          `json:"original_text"`
	Notes        string          `json:"notes"`
	Summary      string          `json:"summary"`
	Category     string          `json:"category"`
	Tags         []string        `json:"tags"`
	Assets       []newsAssetDTO  `json:"assets"`
	CreatedAt    time.Time       `json:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

func normalizeNewsBody(in newsBody) (newsBody, *Error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Source = strings.TrimSpace(in.Source)
	in.URL = strings.TrimSpace(in.URL)
	if in.Title == "" {
		return in, Fail(http.StatusBadRequest, "bad_request", "title is required", nil)
	}
	if in.Source == "" {
		return in, Fail(http.StatusBadRequest, "bad_request", "source is required", nil)
	}
	if in.PublishedAt.IsZero() {
		return in, Fail(http.StatusBadRequest, "bad_request", "published_at is required", nil)
	}
	if in.URL != "" {
		u, err := url.ParseRequestURI(in.URL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return in, Fail(http.StatusBadRequest, "bad_request", "url must be an HTTP(S) URL", nil)
		}
	}
	in.OriginalText = strings.TrimSpace(in.OriginalText)
	in.Notes = strings.TrimSpace(in.Notes)
	in.Summary = strings.TrimSpace(in.Summary)
	in.Category = strings.TrimSpace(in.Category)
	in.Tags = normalizeStrings(in.Tags)
	if in.Assets == nil {
		in.Assets = []newsAssetBody{}
	}
	for i := range in.Assets {
		asset, apiErr := normalizeNewsAsset(in.Assets[i])
		if apiErr != nil {
			return in, apiErr
		}
		in.Assets[i] = asset
	}
	return in, nil
}

func normalizeNewsAsset(in newsAssetBody) (newsAssetBody, *Error) {
	in.AssetType = strings.ToLower(strings.TrimSpace(in.AssetType))
	if in.AssetType != "stock" && in.AssetType != "etf" && in.AssetType != "index" {
		return in, Fail(http.StatusBadRequest, "bad_request", "asset_type must be stock, etf, or index", nil)
	}
	in.Symbol = strings.ToUpper(strings.TrimSpace(in.Symbol))
	if in.Symbol == "" {
		return in, Fail(http.StatusBadRequest, "bad_request", "symbol is required", nil)
	}
	in.Market = strings.TrimSpace(in.Market)
	in.Exchange = strings.TrimSpace(in.Exchange)
	in.DisplayName = strings.TrimSpace(in.DisplayName)
	in.Relation = strings.TrimSpace(in.Relation)
	in.Source = strings.ToLower(strings.TrimSpace(in.Source))
	if in.Source == "" {
		in.Source = "user"
	}
	if in.Source != "user" && in.Source != "ai" {
		return in, Fail(http.StatusBadRequest, "bad_request", "asset source must be user or ai", nil)
	}
	return in, nil
}

func normalizeStrings(items []string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func encodeStrings(items []string) string {
	if items == nil {
		items = []string{}
	}
	b, _ := json.Marshal(items)
	return string(b)
}

func parseStrings(raw string) []string {
	var items []string
	if json.Unmarshal([]byte(raw), &items) != nil || items == nil {
		return []string{}
	}
	return items
}

func toNewsAssetDTO(a store.NewsAsset) newsAssetDTO {
	return newsAssetDTO{
		ID: a.ID, NewsID: a.NewsID, CreatedAt: a.CreatedAt, UpdatedAt: a.UpdatedAt,
		newsAssetBody: newsAssetBody{
			AssetType: a.AssetType, Symbol: a.Symbol, Market: a.Market, Exchange: a.Exchange,
			DisplayName: a.DisplayName, Relation: a.Relation, Source: a.Source,
		},
	}
}

func toNewsDTO(n store.News, assets []store.NewsAsset) newsDTO {
	out := newsDTO{
		ID: n.ID, UserID: n.UserID, CreatedAt: n.CreatedAt, UpdatedAt: n.UpdatedAt,
		Title: n.Title, Source: n.Source, URL: n.Url, PublishedAt: n.PublishedAt,
		OriginalText: n.OriginalText, Notes: n.Notes, Summary: n.Summary,
		Category: n.Category, Tags: parseStrings(n.Tags), Assets: make([]newsAssetDTO, 0, len(assets)),
	}
	for _, asset := range assets {
		out.Assets = append(out.Assets, toNewsAssetDTO(asset))
	}
	return out
}

func newsParams(id, userID string, in newsBody) store.CreateNewsParams {
	return store.CreateNewsParams{
		ID: id, UserID: userID, Title: in.Title, Source: in.Source, Url: in.URL,
		PublishedAt: in.PublishedAt, OriginalText: in.OriginalText, Notes: in.Notes,
		Summary: in.Summary, Category: in.Category, Tags: encodeStrings(in.Tags),
	}
}

func assetParams(id, newsID, userID string, in newsAssetBody) store.CreateNewsAssetParams {
	return store.CreateNewsAssetParams{
		ID: id, NewsID: newsID, UserID: userID, AssetType: in.AssetType, Symbol: in.Symbol,
		Market: in.Market, Exchange: in.Exchange, DisplayName: in.DisplayName,
		Relation: in.Relation, Source: in.Source,
	}
}

func (s *Server) loadNewsDTO(ctx context.Context, q store.Querier, id, userID string) (newsDTO, error) {
	n, err := q.GetNews(ctx, store.GetNewsParams{ID: id, UserID: userID})
	if err != nil {
		return newsDTO{}, err
	}
	assets, err := q.ListNewsAssets(ctx, store.ListNewsAssetsParams{NewsID: id, UserID: userID})
	if err != nil {
		return newsDTO{}, err
	}
	out := toNewsDTO(n, assets)
	out.Predictions, err = loadPredictions(ctx, q, id, userID)
	return out, err
}

func (s *Server) handleCreateNews(c *echo.Context) error {
	var in newsBody
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid body", nil)
	}
	in, apiErr := normalizeNewsBody(in)
	if apiErr != nil {
		return apiErr
	}
	ctx, userID, id := c.Request().Context(), auth.UserID(c), uuid.New().String()
	var out newsDTO
	err := store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		if _, err := q.CreateNews(ctx, newsParams(id, userID, in)); err != nil {
			return err
		}
		for _, asset := range in.Assets {
			if _, err := q.CreateNewsAsset(ctx, assetParams(uuid.New().String(), id, userID, asset)); err != nil {
				return err
			}
		}
		var err error
		out, err = s.loadNewsDTO(ctx, q, id, userID)
		return err
	})
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not create news", nil).Wrap(err)
	}
	return c.JSON(http.StatusCreated, out)
}

func (s *Server) handleListNews(c *echo.Context) error {
	ctx, userID := c.Request().Context(), auth.UserID(c)
	rows, err := s.deps.Store.ListNews(ctx, userID)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not list news", nil).Wrap(err)
	}
	out := make([]newsDTO, 0, len(rows))
	for _, row := range rows {
		assets, err := s.deps.Store.ListNewsAssets(ctx, store.ListNewsAssetsParams{NewsID: row.ID, UserID: userID})
		if err != nil {
			return Fail(http.StatusInternalServerError, "internal", "could not list news", nil).Wrap(err)
		}
		dto := toNewsDTO(row, assets)
		dto.Predictions, err = loadPredictions(ctx, s.deps.Store, row.ID, userID)
		if err != nil {
			return Fail(http.StatusInternalServerError, "internal", "could not load predictions", nil).Wrap(err)
		}
		out = append(out, dto)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) handleGetNews(c *echo.Context) error {
	out, err := s.loadNewsDTO(c.Request().Context(), s.deps.Store, c.Param("id"), auth.UserID(c))
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load news", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) handleUpdateNews(c *echo.Context) error {
	var in newsBody
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid body", nil)
	}
	in, apiErr := normalizeNewsBody(in)
	if apiErr != nil {
		return apiErr
	}
	ctx, id, userID := c.Request().Context(), c.Param("id"), auth.UserID(c)
	_, err := s.deps.Store.UpdateNews(ctx, store.UpdateNewsParams{
		Title: in.Title, Source: in.Source, Url: in.URL, PublishedAt: in.PublishedAt,
		OriginalText: in.OriginalText, Notes: in.Notes, Summary: in.Summary,
		Category: in.Category, Tags: encodeStrings(in.Tags), ID: id, UserID: userID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not update news", nil).Wrap(err)
	}
	out, err := s.loadNewsDTO(ctx, s.deps.Store, id, userID)
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load news", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, out)
}

func (s *Server) handleDeleteNews(c *echo.Context) error {
	n, err := s.deps.Store.DeleteNews(c.Request().Context(), store.DeleteNewsParams{
		ID: c.Param("id"), UserID: auth.UserID(c),
	})
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not delete news", nil).Wrap(err)
	}
	if n == 0 {
		return Fail(http.StatusNotFound, "not_found", "news not found", nil)
	}
	return c.NoContent(http.StatusNoContent)
}

func (s *Server) handleCreateNewsAsset(c *echo.Context) error {
	var in newsAssetBody
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid body", nil)
	}
	in, apiErr := normalizeNewsAsset(in)
	if apiErr != nil {
		return apiErr
	}
	a, err := s.deps.Store.CreateNewsAsset(c.Request().Context(), assetParams(
		uuid.New().String(), c.Param("id"), auth.UserID(c), in,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not create news asset", nil).Wrap(err)
	}
	return c.JSON(http.StatusCreated, toNewsAssetDTO(a))
}

func (s *Server) ownedNewsAsset(ctx context.Context, newsID, assetID, userID string) (store.NewsAsset, error) {
	a, err := s.deps.Store.GetNewsAsset(ctx, store.GetNewsAssetParams{ID: assetID, UserID: userID})
	if err == nil && a.NewsID != newsID {
		return store.NewsAsset{}, sql.ErrNoRows
	}
	return a, err
}

func (s *Server) handleUpdateNewsAsset(c *echo.Context) error {
	var in newsAssetBody
	if err := c.Bind(&in); err != nil {
		return Fail(http.StatusBadRequest, "bad_request", "invalid body", nil)
	}
	in, apiErr := normalizeNewsAsset(in)
	if apiErr != nil {
		return apiErr
	}
	ctx, userID := c.Request().Context(), auth.UserID(c)
	if _, err := s.ownedNewsAsset(ctx, c.Param("id"), c.Param("assetId"), userID); errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news asset not found", nil)
	} else if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load news asset", nil).Wrap(err)
	}
	a, err := s.deps.Store.UpdateNewsAsset(ctx, store.UpdateNewsAssetParams{
		AssetType: in.AssetType, Symbol: in.Symbol, Market: in.Market, Exchange: in.Exchange,
		DisplayName: in.DisplayName, Relation: in.Relation, Source: in.Source,
		ID: c.Param("assetId"), UserID: userID,
	})
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not update news asset", nil).Wrap(err)
	}
	return c.JSON(http.StatusOK, toNewsAssetDTO(a))
}

func (s *Server) handleDeleteNewsAsset(c *echo.Context) error {
	ctx, userID := c.Request().Context(), auth.UserID(c)
	if _, err := s.ownedNewsAsset(ctx, c.Param("id"), c.Param("assetId"), userID); errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news asset not found", nil)
	} else if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not load news asset", nil).Wrap(err)
	}
	if _, err := s.deps.Store.DeleteNewsAsset(ctx, store.DeleteNewsAssetParams{
		ID: c.Param("assetId"), UserID: userID,
	}); err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not delete news asset", nil).Wrap(err)
	}
	return c.NoContent(http.StatusNoContent)
}
