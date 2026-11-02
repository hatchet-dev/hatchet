package workflowruns

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

func TestTriggerErrorResponse(t *testing.T) {
	throttled, err := status.New(codes.ResourceExhausted, "throttled").
		WithDetails(&errdetails.RetryInfo{RetryDelay: durationpb.New(time.Second)})
	require.NoError(t, err)

	t.Run("quota stays a 400", func(t *testing.T) {
		resp, err := triggerErrorResponse(status.Error(codes.ResourceExhausted, "tenant limit reached"))

		require.NoError(t, err)
		require.IsType(t, gen.V1WorkflowRunCreate400JSONResponse{}, resp)
	})

	t.Run("invalid argument is a 400", func(t *testing.T) {
		resp, err := triggerErrorResponse(status.Error(codes.InvalidArgument, "bad"))

		require.NoError(t, err)
		require.IsType(t, gen.V1WorkflowRunCreate400JSONResponse{}, resp)
	})

	t.Run("throttling is returned for the error handler", func(t *testing.T) {
		in := fmt.Errorf("trigger: %w", throttled.Err())

		resp, err := triggerErrorResponse(in)

		require.Nil(t, resp)
		require.ErrorIs(t, err, in)
	})

	t.Run("other errors are returned", func(t *testing.T) {
		resp, err := triggerErrorResponse(fmt.Errorf("boom"))

		require.Nil(t, resp)
		require.Error(t, err)
	})
}
