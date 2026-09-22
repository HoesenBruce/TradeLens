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
