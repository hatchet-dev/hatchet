package run

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

type failingRequestBody struct{ err error }

func (b failingRequestBody) Read([]byte) (int, error) { return 0, b.err }

func TestGeneratedHandlerBindingLogsAndRedactsReadFailure(t *testing.T) {
	cause := &net.OpError{
		Op: "read", Net: "tcp",
		Source: &net.TCPAddr{IP: net.ParseIP("10.0.0.8"), Port: 8080},
		Addr:   &net.TCPAddr{IP: net.ParseIP("192.0.2.1"), Port: 12345},
		Err:    errors.New("i/o timeout"),
	}
	for _, contentType := range []string{echo.MIMEApplicationJSON, echo.MIMEApplicationXML, echo.MIMEApplicationForm} {
		t.Run(contentType, func(t *testing.T) {
			var logs bytes.Buffer
			l := zerolog.New(&logs)
			e := echo.New()
			e.Binder = &requestBinder{Binder: e.Binder, l: &l}
			e.HTTPErrorHandler = retryableLimitErrorHandler(e.DefaultHTTPErrorHandler)
			e.POST("/login", gen.NewStrictHandler(nil).UserUpdateLogin)
			req := httptest.NewRequest(http.MethodPost, "/login", io.NopCloser(failingRequestBody{cause}))
			req.Header.Set(echo.HeaderContentType, contentType)
			out := httptest.NewRecorder()

			e.ServeHTTP(out, req)

			require.Equal(t, http.StatusBadRequest, out.Code)
			require.JSONEq(t, `{"message":"Bad Request"}`, out.Body.String())
			require.Contains(t, logs.String(), "10.0.0.8:8080")
			require.Contains(t, logs.String(), "i/o timeout")
		})
	}
}

func TestRequestBinderPreservesUnsupportedMediaType(t *testing.T) {
	l := zerolog.Nop()
	e := echo.New()
	binder := &requestBinder{Binder: e.Binder, l: &l}
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("body"))
	req.Header.Set(echo.HeaderContentType, "application/octet-stream")
	ctx := e.NewContext(req, httptest.NewRecorder())
	var body map[string]interface{}
	err := binder.Bind(&body, ctx)
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnsupportedMediaType, httpErr.Code)
}
