package server

import (
	"net/http"

	"github.com/labstack/echo/v4"

	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

// VersionInfo returns the current version information.
func (e *EverestServer) VersionInfo(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, &api.Version{
		ProjectName: version.ProjectName,
		Version:     version.Version,
		FullCommit:  version.FullCommit,
	})
}
