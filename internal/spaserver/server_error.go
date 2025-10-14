package spaserver

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// обработка ошибки
func (s *Server) ServerError(c echo.Context, err error) error {
	return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
}
