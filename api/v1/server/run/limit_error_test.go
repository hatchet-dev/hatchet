package run

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/fairpool"
)

func throttledStatus(t *testing.T, retryAfter time.Duration) error {
	t.Helper()

	st, err := status.New(codes.ResourceExhausted, "throttled").WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(retryAfter)})
	require.NoError(t, err)

	return st.Err()
}

func TestRetryableLimitErrorHandler(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = retryableLimitErrorHandler(e.DefaultHTTPErrorHandler)

	for name, tc := range map[string]struct {
		err        error
		retryAfter string
		want       int
	}{
		"limit error":                    {err: fmt.Errorf("trigger: %w", &fairpool.LimitError{Key: "shared", Limit: 2}), want: http.StatusTooManyRequests, retryAfter: "1"},
		"proxied throttle":               {err: throttledStatus(t, 2*time.Second), want: http.StatusTooManyRequests, retryAfter: "2"},
		"proxied throttle rounds up":     {err: throttledStatus(t, 1500*time.Millisecond), want: http.StatusTooManyRequests, retryAfter: "2"},
		"proxied throttle without delay": {err: throttledStatus(t, 0), want: http.StatusTooManyRequests, retryAfter: "1"},
		"proxied resource exhausted":     {err: status.Error(codes.ResourceExhausted, "limit"), want: http.StatusInternalServerError},
		"proxied unavailable":            {err: status.Error(codes.Unavailable, "down"), want: http.StatusInternalServerError},
		"other error":                    {err: fmt.Errorf("boom"), want: http.StatusInternalServerError},
		"explicit http error":            {err: echo.NewHTTPError(http.StatusBadRequest, "bad"), want: http.StatusBadRequest},
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			e.HTTPErrorHandler(tc.err, c)

			require.Equal(t, tc.want, rec.Code)
			require.Equal(t, tc.retryAfter, rec.Header().Get(echo.HeaderRetryAfter))

			if tc.want != http.StatusTooManyRequests {
				return
			}

			require.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)

			var body gen.APIErrors
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Len(t, body.Errors, 1)
			require.Equal(t, "too many concurrent database operations, retry shortly", body.Errors[0].Description)
		})
	}
}

func TestRetryableLimitErrorHandlerCommittedResponse(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = retryableLimitErrorHandler(e.DefaultHTTPErrorHandler)

	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)
	require.NoError(t, c.String(http.StatusOK, "partial"))

	e.HTTPErrorHandler(&fairpool.LimitError{Key: "shared", Limit: 2}, c)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "partial", rec.Body.String())
	require.Empty(t, rec.Header().Get(echo.HeaderRetryAfter))
}

// The access logger must see the status the handler wrote and still receive the original error.
func TestThrottledRequestIsLoggedAs429(t *testing.T) {
	e := echo.New()
	e.HTTPErrorHandler = retryableLimitErrorHandler(e.DefaultHTTPErrorHandler)

	cause := fmt.Errorf("trigger: %w", &fairpool.LimitError{Key: "shared", Limit: 2})

	var logged middleware.RequestLoggerValues

	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError: true,
		LogStatus:   true,
		LogError:    true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			logged = v
			return nil
		},
	}))
	e.GET("/", func(echo.Context) error { return cause })

	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	require.Equal(t, http.StatusTooManyRequests, rec.Code)
	require.Equal(t, http.StatusTooManyRequests, logged.Status)
	require.ErrorIs(t, logged.Error, cause)
}
