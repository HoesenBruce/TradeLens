package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v5"
	"github.com/tradermemos/api/internal/auth"
	"github.com/tradermemos/api/internal/exporter"
	"github.com/tradermemos/api/internal/store"
)

func (s *Server) handleNewsExport(c *echo.Context) error {
	ctx := c.Request().Context()
	var input exporter.NewsInput
	err := store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		var err error
		input, err = exporter.LoadNews(ctx, q, auth.UserID(c), c.Param("id"))
		return err
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "news not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not export news", nil).Wrap(err)
	}
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, exporter.NewsFilename(input.News)))
	return c.Blob(http.StatusOK, "text/markdown; charset=utf-8", []byte(exporter.NewsMarkdown(input)))
}

func (s *Server) handleNewsBatchExport(c *echo.Context) error {
	f, err := parseNewsPerformanceFilter(c)
	if err != nil {
		return err
	}
	// Repeated id parameters allow a client to export its exact selected list.
	selected := map[string]bool{}
	for _, id := range c.QueryParams()["id"] {
		if id == "" {
			return Fail(http.StatusBadRequest, "bad_request", "id must not be empty", nil)
		}
		selected[id] = false
	}
	ctx, owner := c.Request().Context(), auth.UserID(c)
	var inputs []exporter.NewsInput
	err = store.InTx(ctx, s.deps.Store, func(q store.Querier) error {
		rows, err := q.ListNews(ctx, owner)
		if err != nil {
			return err
		}
		// ponytail: per-thesis reads suit personal journals; batch queries if export volume warrants it.
		for _, n := range rows {
			if len(selected) > 0 {
				if _, ok := selected[n.ID]; !ok {
					continue
				}
				selected[n.ID] = true
			}
			if !f.MatchesNews(n) {
				continue
			}
			input, err := exporter.LoadNews(ctx, q, owner, n.ID)
			if err != nil {
				return err
			}
			if exporter.MatchesNews(input, f) {
				inputs = append(inputs, input)
			}
		}
		for _, found := range selected {
			if !found {
				return sql.ErrNoRows
			}
		}
		return nil
	})
	if errors.Is(err, sql.ErrNoRows) {
		return Fail(http.StatusNotFound, "not_found", "selected news not found", nil)
	}
	if err != nil {
		return Fail(http.StatusInternalServerError, "internal", "could not export news", nil).Wrap(err)
	}
	c.Response().Header().Set("Content-Disposition", `attachment; filename="news-theses.md"`)
	return c.Blob(http.StatusOK, "text/markdown; charset=utf-8", []byte(exporter.NewsBatchMarkdown(inputs)))
}
