package run

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
	"github.com/hatchet-dev/hatchet/pkg/repository"
)

func TestAccessLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		err       error
		wantLevel zerolog.Level
	}{
		{
			name:      "client closed request logs at info",
			status:    hatchetmiddleware.StatusClientClosedRequest,
			wantLevel: zerolog.InfoLevel,
		},
		{
			name:      "server error logs at error",
			status:    http.StatusInternalServerError,
			wantLevel: zerolog.ErrorLevel,
		},
		{
			name:      "client error is a warning",
			status:    http.StatusForbidden,
			wantLevel: zerolog.WarnLevel,
		},
		{
			name:      "refusal by an operator control is debug",
			status:    http.StatusTooManyRequests,
			err:       fmt.Errorf("could not ingest event: %w", &repository.TenantRateLimitedError{RetryAfter: time.Second}),
			wantLevel: zerolog.DebugLevel,
		},
		{
			name:      "a paused tenant's refusal is debug",
			status:    http.StatusTooManyRequests,
			err:       fmt.Errorf("could not ingest event: %w", repository.ErrTenantPaused),
			wantLevel: zerolog.DebugLevel,
		},
		{
			name:      "a paused tenant's proxied refusal is debug",
			status:    http.StatusTooManyRequests,
			err:       status.Error(codes.ResourceExhausted, repository.ErrTenantPaused.Error()),
			wantLevel: zerolog.DebugLevel,
		},
		{
			name:      "other throttling is a warning",
			status:    http.StatusTooManyRequests,
			err:       fmt.Errorf("too many requests"),
			wantLevel: zerolog.WarnLevel,
		},
		{
			name:      "success is informational",
			status:    http.StatusOK,
			wantLevel: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := accessLogLevel(tt.status, tt.err)

			if level != tt.wantLevel {
				t.Errorf("level = %s, want %s", level, tt.wantLevel)
			}
		})
	}
}
