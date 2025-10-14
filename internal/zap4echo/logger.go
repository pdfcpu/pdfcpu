package zap4echo

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const DefaultLoggerMsg = "Served"
const DefaultRequestIDHeader = echo.HeaderXRequestID

var defaultLoggerConfig = LoggerConfig{}

type LoggerConfig struct {
	// Only log requests that respond with a status code of
	// 3XX, 4XX, or 5XX, or when the handler returns an error.
	ErrorOnly bool

	// Skip the current request depending on the context.
	Skipper func(c echo.Context) bool

	// Custom string for the `msg` field
	CustomMsg string

	// Don't omit the `caller` field. By default, caller will not be printed.
	//
	// Caller gets printed as `zap4echo/logger.go:121`. That is redundant.
	IncludeCaller bool

	// If true, printing of stack trace will be disabled.
	OmitStackTrace bool

	// If true, particular field will not be printed.
	OmitStatusText bool
	OmitClientIP   bool
	OmitUserAgent  bool
	OmitPath       bool
	OmitRequestID  bool
	OmitReferer    bool

	// Custom header name for request ID
	CustomRequestIDHeader string

	// A function for adding custom fields depending on the context.
	FieldAdder func(c echo.Context) []zapcore.Field
}

func Logger(log *zap.Logger) echo.MiddlewareFunc {
	return LoggerWithConfig(log, defaultLoggerConfig)
}

func LoggerWithConfig(log *zap.Logger, config LoggerConfig) echo.MiddlewareFunc {
	if !config.IncludeCaller {
		log = log.WithOptions(zap.WithCaller(false))
	}

	if config.OmitStackTrace {
		log = log.WithOptions(zap.AddStacktrace(zap.FatalLevel + 1))
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			herr := next(c)
			if herr != nil {
				c.Error(herr)
			}

			if config.Skipper != nil && config.Skipper(c) {
				return nil
			}

			resp := c.Response()
			req := c.Request()

			if config.ErrorOnly && (resp.Status < 300 && herr == nil) {
				return nil
			}

			latency := time.Since(start)
			fields := make([]zapcore.Field, 0, 15)

			fields = append(fields, []zapcore.Field{
				zap.String("proto", req.Proto),
				zap.String("host", req.Host),
				zap.String("method", req.Method),
				zap.Int("status", resp.Status),
				zap.Int64("response_size", resp.Size),
				zap.Duration("latency", latency),
			}...)

			if !config.OmitStatusText {
				fields = append(fields, zap.String("status_text", http.StatusText(resp.Status)))
			}

			if !config.OmitClientIP {
				fields = append(fields, zap.String("client_ip", c.RealIP()))
			}

			if !config.OmitUserAgent {
				fields = append(fields, zap.String("user_agent", req.UserAgent()))
			}

			if !config.OmitPath {
				// Use RequestURI instead of URL.Path.
				// See: https://github.com/golang/go/issues/2782
				fields = append(fields, zap.String("path", req.RequestURI))
			}

			if !config.OmitRequestID {
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
			}

			if !config.OmitReferer {
				referer := resp.Writer.Header().Get("Referer")
				if referer == "" {
					referer = req.Header.Get("Referer")
				}
				if referer != "" {
					fields = append(fields, zap.String("referer", referer))
				}
			}

			if config.FieldAdder != nil {
				fields = append(fields, config.FieldAdder(c)...)
			}
			if herr != nil {
				fields = append(fields, zap.Error(herr))
			}

			s := resp.Status
			msg := func() string {
				if config.CustomMsg == "" {
					return DefaultLoggerMsg
				} else {
					return config.CustomMsg
				}
			}()
			switch {
			case s >= 500:
				log.Error(msg, fields...)
			case s >= 400:
				log.Warn(msg, fields...)
			case s >= 300:
				log.Info(msg, fields...)
			default:
				log.Info(msg, fields...)
			}

			// We already handled error with c.Error
			return nil
		}
	}
}
