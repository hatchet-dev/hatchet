// Deprecated: This package is part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
package client

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	dispatchercontracts "github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/retry"
)

func TestWorkflowResultSingleAddWorkflowRunAttempt(t *testing.T) {
	logger := zerolog.Nop()
	constructorCalls := atomic.Int32{}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		constructorCalls.Add(1)
		return nil, status.Error(codes.Unavailable, "engine down")
	}, nil)

	workflow := NewWorkflow("run-1", listener)
	_, err := workflow.Result()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to listen for workflow events")
	assert.LessOrEqual(t, constructorCalls.Load(), int32(retry.StreamSyncMaxAttempts))
	assert.Greater(t, constructorCalls.Load(), int32(0))

	assert.False(t, listener.reg.hasAny())
}

func TestWorkflowResultFailsWhenListenerDiesPermanently(t *testing.T) {
	logger := zerolog.Nop()
	runID := "run-permanent-fail"

	client := &mockSubscribeClient{
		recvFn: func() (*dispatchercontracts.WorkflowRunEvent, error) {
			return nil, status.Error(codes.Unauthenticated, "auth failed")
		},
		recvChan: make(chan *dispatchercontracts.WorkflowRunEvent),
	}

	listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
		return client, nil
	}, client)

	workflow := NewWorkflow(runID, listener)

	resultDone := make(chan struct{})
	var resultErr error
	go func() {
		_, resultErr = workflow.Result()
		close(resultDone)
	}()

	require.Eventually(t, func() bool {
		select {
		case <-resultDone:
			return true
		default:
			return false
		}
	}, 5*time.Second, 10*time.Millisecond)

	require.Error(t, resultErr)
	assert.Contains(t, resultErr.Error(), runID)
	assert.ErrorIs(t, resultErr, status.Error(codes.Unauthenticated, "auth failed"))

	require.NoError(t, listener.Close())
}

func TestWorkflowResultPollsWhenListenerIsSilent(t *testing.T) {
	type fetchResponse struct {
		details *RunDetails
		err     error
	}

	tests := []struct {
		name          string
		responses     func(uuid.UUID) []fetchResponse
		wantFetches   int
		wantError     string
		wantStepValue string
	}{
		{
			name: "transient error and running before completed",
			responses: func(runID uuid.UUID) []fetchResponse {
				emptyError := ""
				return []fetchResponse{
					{err: errors.New("temporary fetch failure")},
					{details: &RunDetails{
						ExternalId: runID,
						Status:     rest.V1TaskStatusRUNNING,
						Done:       false,
					}},
					{details: &RunDetails{
						ExternalId: runID,
						Status:     rest.V1TaskStatusCOMPLETED,
						Done:       true,
						TaskRuns: map[string]*TaskRunDetails{
							"step-1": {
								ExternalId: uuid.New(),
								ReadableId: "step-1",
								Status:     rest.V1TaskStatusCOMPLETED,
								Output:     json.RawMessage(`{"value":"complete"}`),
								Error:      &emptyError,
							},
						},
					}},
				}
			},
			wantFetches:   3,
			wantStepValue: "complete",
		},
		{
			name: "failed run returns task error",
			responses: func(runID uuid.UUID) []fetchResponse {
				taskError := "step failed"
				return []fetchResponse{
					{details: &RunDetails{
						ExternalId: runID,
						Status:     rest.V1TaskStatusFAILED,
						Done:       true,
						TaskRuns: map[string]*TaskRunDetails{
							"step-1": {
								ExternalId: uuid.New(),
								ReadableId: "step-1",
								Status:     rest.V1TaskStatusFAILED,
								Error:      &taskError,
							},
						},
					}},
				}
			},
			wantFetches: 1,
			wantError:   "step failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zerolog.Nop()
			recvChan := make(chan *dispatchercontracts.WorkflowRunEvent)
			streamClient := &mockSubscribeClient{recvChan: recvChan}
			listener := newTestWorkflowRunsListener(t, &logger, func(ctx context.Context) (dispatchercontracts.Dispatcher_SubscribeToWorkflowRunsClient, error) {
				return streamClient, nil
			}, streamClient)
			t.Cleanup(func() {
				close(recvChan)
				require.NoError(t, listener.Close())
			})

			runID := uuid.New()
			responses := tt.responses(runID)
			fetchCalls := 0
			workflow := NewWorkflow(runID.String(), listener, func(ctx context.Context, id uuid.UUID) (*RunDetails, error) {
				assert.Equal(t, runID, id)
				response := responses[fetchCalls]
				fetchCalls++
				return response.details, response.err
			})
			workflow.resultPollGrace = time.Millisecond
			workflow.resultPollInterval = time.Millisecond

			type result struct {
				value *WorkflowResult
				err   error
			}
			resultChan := make(chan result, 1)
			go func() {
				value, err := workflow.Result()
				resultChan <- result{value: value, err: err}
			}()

			var got result
			select {
			case got = <-resultChan:
			case <-time.After(5 * time.Second):
				t.Fatal("timed out waiting for workflow result fallback")
			}

			require.Equal(t, tt.wantFetches, fetchCalls)
			if tt.wantError != "" {
				require.EqualError(t, got.err, tt.wantError)
				return
			}

			require.NoError(t, got.err)
			require.NotNil(t, got.value)
			var stepOutput struct {
				Value string `json:"value"`
			}
			require.NoError(t, got.value.StepOutput("step-1", &stepOutput))
			assert.Equal(t, tt.wantStepValue, stepOutput.Value)
		})
	}
}
