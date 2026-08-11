package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func TestSessionSaveError(t *testing.T) {
	logger := zerolog.Nop()
	a := &AuthN{l: &logger}
	saveErr := errors.New("connection reset by peer")

	t.Run("canceled request maps to 499", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := a.sessionSaveError(ctx, saveErr)

		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != middleware.StatusClientClosedRequest {
			t.Fatalf("expected HTTP %d, got %v", middleware.StatusClientClosedRequest, err)
		}

		if !errors.Is(err, saveErr) {
			t.Errorf("save error not preserved: %v", err)
		}
	})

	t.Run("expired request deadline maps to 499", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
		defer cancel()

		err := a.sessionSaveError(ctx, saveErr)

		var httpErr *echo.HTTPError
		if !errors.As(err, &httpErr) || httpErr.Code != middleware.StatusClientClosedRequest {
			t.Fatalf("expected HTTP %d, got %v", middleware.StatusClientClosedRequest, err)
		}

		if !errors.Is(err, saveErr) {
			t.Errorf("save error not preserved: %v", err)
		}
	})

	t.Run("ordinary save failure on a live request stays a server error", func(t *testing.T) {
		err := a.sessionSaveError(context.Background(), saveErr)

		if err == nil {
			t.Fatal("expected an error")
		}

		var httpErr *echo.HTTPError
		if errors.As(err, &httpErr) {
			t.Fatalf("expected a plain server error, got HTTP %d", httpErr.Code)
		}
	})
}
