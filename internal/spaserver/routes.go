package spaserver

import (
	"bytes"
	"image/png"
	"net/http"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/labstack/echo/v4"
	"github.com/mechiko/utility"
)

// маршрутизация приложения
func (s *Server) Routes() error {
	s.server.GET("/123.png", s.Datamatrix)
	return nil
}

const cisraw = `0105000213100066215aDos=X93a2MS`

func (s *Server) Datamatrix(c echo.Context) error {
	cis, err := utility.ParseCisInfo(cisraw)
	if err != nil {
		return s.ServerError(c, err)
	}
	cisFnc := cis.FNC1()
	qrCode, _ := datamatrix.Encode(cisFnc)
	qrCode, _ = barcode.Scale(qrCode, 100, 100)
	var b bytes.Buffer
	png.Encode(&b, qrCode)
	return c.Stream(http.StatusOK, "image/png", &b)
}
