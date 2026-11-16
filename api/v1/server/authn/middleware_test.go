package authn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func TestSessionKeepsUserUnverified(t *testing.T) {
	tests := []struct {
		name  string
		value interface{}
		set   bool
		want  bool
	}{
		{name: "session without a value follows the user", set: false, want: false},
		{name: "verified", value: "true", set: true, want: false},
		{name: "verified bool", value: true, set: true, want: false},
		{name: "unverified", value: "false", set: true, want: true},
		{name: "unverified bool", value: false, set: true, want: true},
		{name: "unknown value counts as unverified", value: 1, set: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session := sessions.NewSession(nil, "test")
			if tt.set {
				session.Values[sessionEmailVerifiedKey] = tt.value
			}

			if got := sessionKeepsUserUnverified(session); got != tt.want {
				t.Fatalf("sessionKeepsUserUnverified() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
