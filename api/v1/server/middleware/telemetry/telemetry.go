package telemetry

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel"

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
