//go:build !e2e && !load && !rampup && !integration

package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/client"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
)

type stubDispatcher struct {
	client.DispatcherClient
}

func (stubDispatcher) SendStepActionEvent(context.Context, *client.ActionEvent) (*client.ActionEventResponse, error) {
	return &client.ActionEventResponse{}, nil
}

type stubClient struct {
	client.Client
}

func (stubClient) Dispatcher() client.DispatcherClient { return stubDispatcher{} }
func (stubClient) Namespace() string                   { return "" }

type ctxKey struct{}

type observed struct {
	err             error
	cause           error
	derivedCause    error
	derivedHasValue bool
}

// runUntilCancelled starts a step run through startStepRun, with a middleware that
// replaces the task context via SetContext, and interrupts it with cancel once the
// handler is running.
func runUntilCancelled(t *testing.T, cancel func(w *Worker, stepRunId string)) observed {
	t.Helper()

	l := zerolog.Nop()
	w := &Worker{
		client:      stubClient{},
		actions:     ActionRegistry{},
		l:           &l,
		alerter:     hatcheterrors.NoOpAlerter{},
		middlewares: newMiddlewares(),
	}

	// Mirrors the OTel middleware, which wraps the task context and installs it with SetContext.
	w.middlewares.add(func(ctx HatchetContext, next func(HatchetContext) error) error {
		ctx.SetContext(context.WithValue(ctx.GetContext(), ctxKey{}, "derived"))
		return next(ctx)
	})

	started := make(chan struct{})
	result := make(chan observed, 1)

	if err := w.RegisterAction("svc:verb", func(ctx HatchetContext) error {
		close(started)
		<-ctx.Done()
		result <- observed{
			err:             ctx.Err(),
			cause:           context.Cause(ctx),
			derivedCause:    context.Cause(ctx.GetContext()),
			derivedHasValue: ctx.GetContext().Value(ctxKey{}) == "derived",
		}
		return nil
	}); err != nil {
		t.Fatalf("register action: %v", err)
	}

	const stepRunId = "step-run-1"
	done := make(chan error, 1)

	go func() {
		done <- w.startStepRun(context.Background(), &client.Action{
			ActionId:   "svc:verb",
			ActionType: client.ActionTypeStartStepRun,
			StepRunId:  stepRunId,
		})
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("handler did not start")
	}

	cancel(w, stepRunId)

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("startStepRun: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("startStepRun did not return after cancel")
	}

	return <-result
}

func TestStepRunCancelCause(t *testing.T) {
	tests := []struct {
		name    string
		cancel  func(w *Worker, stepRunId string)
		want    error
		notWant error
	}{
		{
			name: "cancelled",
			cancel: func(w *Worker, stepRunId string) {
				if err := w.cancelStepRun(context.Background(), &client.Action{StepRunId: stepRunId}); err != nil {
					t.Errorf("cancelStepRun: %v", err)
				}
			},
			want:    ErrTaskCancelled,
			notWant: ErrTaskEvicted,
		},
		{
			name: "evicted",
			cancel: func(w *Worker, stepRunId string) {
				w.CancelStepRun(stepRunId)
			},
			want:    ErrTaskEvicted,
			notWant: ErrTaskCancelled,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := runUntilCancelled(t, tt.cancel)

			if !errors.Is(got.err, context.Canceled) {
				t.Errorf("ctx.Err() = %v, want context.Canceled", got.err)
			}

			if !got.derivedHasValue {
				t.Fatal("handler did not receive the context installed by middleware")
			}

			for label, cause := range map[string]error{
				"context.Cause(hctx)":              got.cause,
				"context.Cause(hctx.GetContext())": got.derivedCause,
			} {
				if !errors.Is(cause, tt.want) {
					t.Errorf("%s = %v, want %v", label, cause, tt.want)
				}
				if errors.Is(cause, tt.notWant) {
					t.Errorf("%s = %v, must not match %v", label, cause, tt.notWant)
				}
			}
		})
	}
}
