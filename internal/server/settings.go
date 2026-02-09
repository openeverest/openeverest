package server

import (
	"net/http"

	"github.com/AlekSi/pointer"
	"github.com/labstack/echo/v4"

	"github.com/openeverest/openeverest/v2/api"
)

// GetSettings returns the Everest global settings.
func (e *EverestServer) GetSettings(ctx echo.Context) error {
	result, err := e.handler.GetSettings(ctx.Request().Context())
	if err != nil {
		return ctx.JSON(http.StatusInternalServerError, api.Error{
			Message: pointer.To("Failed to get Everest settings"),
		})
	}
	return ctx.JSON(http.StatusOK, result)
}
