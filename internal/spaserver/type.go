package spaserver

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/mechiko/pdfcpu/internal/zap4echo"
	"go.uber.org/zap"
)

const (
	// _defaultReadTimeout     = 5 * time.Second
	// _defaultWriteTimeout    = 5 * time.Second
	_defaultAddr            = "127.0.0.1:8888"
	_defaultShutdownTimeout = 5 * time.Second
)

// Server -.
type Server struct {
	server          *echo.Echo
	addr            string
	notify          chan error
	shutdownTimeout time.Duration
	debug           bool
}

// var sseManager *sse.Server

func New(host string, port string) (ss *Server, err error) {
	addr := fmt.Sprintf("%s:%s", host, port)
	if port == "" {
		addr = _defaultAddr
	}
	e := echo.New()
	e.Logger.SetOutput(io.Discard)
	log, _ := zap.NewDevelopment()

	e.Use(
		zap4echo.Logger(log),
		zap4echo.Recover(log),
	)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		// AllowHeaders:     []string{"authorization", "Content-Type"},
		AllowHeaders:     []string{echo.HeaderContentType, echo.HeaderAuthorization, echo.HeaderXCSRFToken},
		AllowCredentials: true,
		AllowMethods:     []string{http.MethodGet, http.MethodPost},
		// AllowMethods:     []string{echo.OPTIONS, echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))

	ss = &Server{
		server:          e,
		addr:            addr,
		notify:          make(chan error, 1),
		shutdownTimeout: _defaultShutdownTimeout,
		debug:           true,
	}

	if err := ss.Routes(); err != nil {
		return nil, fmt.Errorf("spaserver new routes error %w", err)
	}
	return ss, nil
}

func (s *Server) Start() {
	go func() {
		// s.notify <- s.server.StartTLS(s.addr, "cert.pem", "key.pem")
		s.notify <- s.server.Start(s.addr)
		close(s.notify)
	}()
}

func (s *Server) Notify() <-chan error {
	return s.notify
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	err := s.server.Shutdown(ctx)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Echo() *echo.Echo {
	return s.server
}
