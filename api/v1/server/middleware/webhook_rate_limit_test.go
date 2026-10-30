package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestRetryAfter(t *testing.T) {
	require.Equal(t, "1", RetryAfter(50))
	require.Equal(t, "1", RetryAfter(1))
	require.Equal(t, "2", RetryAfter(0.5))
	require.Equal(t, "4", RetryAfter(0.3))
	require.Equal(t, "1", RetryAfter(0))
}

func TestWebhookRateLimitSetsRetryAfter(t *testing.T) {
	logger := zerolog.Nop()
	mw := WebhookRateLimitMiddleware(0.5, 1, &logger)
	handler := func(c echo.Context) error { return c.NoContent(http.StatusOK) }

	e := echo.New()

	var rec *httptest.ResponseRecorder

	for range 2 {
		rec = httptest.NewRecorder()
		c := e.NewContext(httptest.NewRequest(http.MethodPost, "/api/v1/stable/tenants/t/webhooks/w", nil), rec)
		c.SetParamNames("tenant", "v1-webhook")
		c.SetParamValues("t", "w")

		require.NoError(t, mw(handler)(c))
	}

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, "2", rec.Header().Get(echo.HeaderRetryAfter))
}
