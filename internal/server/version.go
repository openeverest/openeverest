// Copyright (C) 2026 The OpenEverest Contributors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
