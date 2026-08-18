package run

import (
	"net/http"
	"testing"

	"github.com/rs/zerolog"

	hatchetmiddleware "github.com/hatchet-dev/hatchet/api/v1/server/middleware"
)

func TestAccessLogLevel(t *testing.T) {
	tests := []struct {
		name      string
		status    int
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
			name:      "success is informational",
			status:    http.StatusOK,
			wantLevel: zerolog.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			level := accessLogLevel(tt.status)

			if level != tt.wantLevel {
				t.Errorf("level = %s, want %s", level, tt.wantLevel)
			}
		})
	}
}
