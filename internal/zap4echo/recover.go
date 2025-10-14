package zap4echo

import (
	"fmt"
	"runtime"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const DefaultRecoverMsg = "Recovered"

var defaultRecoverConfig = RecoverConfig{
	StackTrace:     false,
	StackTraceSize: 4 << 10, // 4 KB
}

type RecoverConfig struct {
	// Custom string for the `msg` field
	CustomMsg string

	// Set this to true to enable stack trace.
	// `stacktrace` field will be used to print stack trace.
	StackTrace bool
	// Size allocated on memory for stack trace.
	StackTraceSize int
	// If stack trace is enabled, this is to print stack traces of all goroutines.
	PrintStackTraceOfAllGoroutines bool

	// Custom header name for request ID
	CustomRequestIDHeader string

	// A function for adding custom fields depending on the context.
	FieldAdder func(c echo.Context, err error) []zap.Field

	// The panic was happened, and it was handled and logged gracefully.
	// What's next?
	//
	// This function is called to handle the error of panic.
	ErrorHandler func(c echo.Context, err error)
}

func Recover(log *zap.Logger) echo.MiddlewareFunc {
	return RecoverWithConfig(log, defaultRecoverConfig)
}

func RecoverWithConfig(log *zap.Logger, config RecoverConfig) echo.MiddlewareFunc {
	if log == nil {
		log = zap.NewNop()
	}
	if config.StackTrace {
		// Disable printing of stacktrace. We will manually print it.
		log = log.WithOptions(zap.AddStacktrace(zap.FatalLevel + 1))

		if config.StackTraceSize == 0 {
			config.StackTraceSize = defaultRecoverConfig.StackTraceSize
		}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			defer func() {
				if err := recover(); err != nil {
					e := func() error {
						if e, ok := err.(error); ok {
							return e
						} else {
							return fmt.Errorf("panic: %v", err)
						}
					}()

					c.Error(e)

					req := c.Request()
					resp := c.Response()

					fields := make([]zap.Field, 0, 6)
					fields = append(fields, []zapcore.Field{
						zap.Any("error", err),
						zap.String("method", req.Method),

						// Use RequestURI instead of URL.Path.
						// See: https://github.com/golang/go/issues/2782
						zap.String("path", req.RequestURI),

						zap.String("client_ip", c.RealIP()),
					}...)

					if config.StackTrace {
						stack := make([]byte, config.StackTraceSize)
						stackLen := runtime.Stack(stack, config.PrintStackTraceOfAllGoroutines)
						fields = append(fields, zap.ByteString("stacktrace", stack[:stackLen]))
					}

					requestIDHeader := func() string {
						if config.CustomRequestIDHeader != "" {
							return config.CustomRequestIDHeader
						} else {
							return DefaultRequestIDHeader
						}
					}()
					requestID := req.Header.Get(requestIDHeader)
					if requestID == "" {
						requestID = resp.Header().Get(requestIDHeader)
					}
					if requestID != "" {
						fields = append(fields, zap.String("request_id", requestID))
					}

					if config.FieldAdder != nil {
						fields = append(fields, config.FieldAdder(c, e)...)
					}

					msg := func() string {
						if config.CustomMsg == "" {
							return DefaultRecoverMsg
						} else {
							return config.CustomMsg
						}
					}()
					log.Error(msg, fields...)

					if config.ErrorHandler != nil {
						config.ErrorHandler(c, e)
					}
				}
			}()
			return next(c)
		}
	}
}
