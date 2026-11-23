package server

import (
	"net/http"
	"slices"

	"github.com/labstack/echo/v4"

	"github.com/openeverest/openeverest/v2/contract"
	api "github.com/openeverest/openeverest/v2/internal/server/api"
	"github.com/openeverest/openeverest/v2/pkg/version"
)

// VersionInfo returns the current version information, with the window of
// contract versions this server was built with.
func (e *EverestServer) VersionInfo(ctx echo.Context) error {
	return ctx.JSON(http.StatusOK, &api.Version{
		ProjectName:        version.ProjectName,
		Version:            version.Version,
		FullCommit:         version.FullCommit,
		ContractVersion:    contract.Version,
		MinContractVersion: contract.MinVersion,
		UpgradableFrom:     slices.Clone(contract.UpgradableFrom),
	})
}
