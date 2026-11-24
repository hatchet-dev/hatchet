package hatchet

import (
	"context"
	"errors"
	"testing"

	"github.com/hatchet-dev/hatchet/pkg/worker"
)

func TestTaskCancelCauseSentinels(t *testing.T) {
	for _, tt := range []struct {
		name          string
		workerCause   error
		want, notWant error
	}{
		{"cancelled", worker.ErrTaskCancelled, ErrTaskCancelled, ErrTaskEvicted}, //nolint:staticcheck // SA1019
		{"evicted", worker.ErrTaskEvicted, ErrTaskEvicted, ErrTaskCancelled},     //nolint:staticcheck // SA1019
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithCancelCause(context.Background())
			cancel(tt.workerCause)

			cause := context.Cause(ctx)
			if !errors.Is(cause, tt.want) {
				t.Errorf("context.Cause = %v, want %v", cause, tt.want)
			}
			if errors.Is(cause, tt.notWant) {
				t.Errorf("context.Cause = %v, must not match %v", cause, tt.notWant)
			}
		})
	}
}
