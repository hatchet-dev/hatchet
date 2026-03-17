package telemetry

import (
	"errors"
	"fmt"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/hatchet-dev/hatchet/pkg/config/server"
)

type OTelMiddleware struct {
	config *server.ServerConfig
}

func NewOTelMiddleware(config *server.ServerConfig) *OTelMiddleware {
	return &OTelMiddleware{
		config: config,
	}
}

func (m *OTelMiddleware) Middleware() echo.MiddlewareFunc {
	serviceName := m.config.OpenTelemetry.ServiceName
	tracerProvider := otel.GetTracerProvider()

	return otelecho.Middleware(serviceName,
		otelecho.WithSkipper(func(c echo.Context) bool {
			path := c.Path()
			return path == "/api/ready" || path == "/api/live"
		}),
		otelecho.WithTracerProvider(tracerProvider),
	)
}

// ErrorStatusMiddleware marks the current span as Error for any 4xx or 5xx response.
// otelecho only sets Error for 5xx (per OTel semantic conventions). This middleware
// must be registered after otelecho so it runs inside the span. The OTel SDK ignores
// attempts to downgrade from Error to Unset, so otelecho's subsequent status-setting
// for 4xx is a no-op.
func (m *OTelMiddleware) ErrorStatusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)

			span := trace.SpanFromContext(c.Request().Context())

			statusCode := 0
			if err != nil {
				var he *echo.HTTPError
				if errors.As(err, &he) {
					statusCode = he.Code
				}
			}

			if statusCode == 0 {
				statusCode = c.Response().Status
			}

			if statusCode >= 400 {
				span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
				if err != nil {
					span.RecordError(err)
				}
			}

			return err
		}
	}
}
