package middleware

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	v1contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type countingAlerter struct {
	n int
}

func (c *countingAlerter) SendAlert(ctx context.Context, err error, data map[string]interface{}) {
	c.n++
}

// runThroughErrorInterceptor returns what a unary and a streaming handler failing with
// handlerErr hand back to the client. The two paths share one hook, so they must agree.
func runThroughErrorInterceptor(t *testing.T, handlerErr error) (error, *countingAlerter) {
	t.Helper()

	alerter := &countingAlerter{}
	l := zerolog.Nop()
	interceptor := NewErrorInterceptor(alerter, &l).Interceptor()

	streamErr := interceptor.WrapStreamingHandler(func(context.Context, connect.StreamingHandlerConn) error {
		return handlerErr
	})(context.Background(), stubHandlerConn{})

	streamAlerts := alerter.n
	alerter.n = 0

	_, unaryErr := interceptor.WrapUnary(func(context.Context, connect.AnyRequest) (connect.AnyResponse, error) {
		return nil, handlerErr
	})(context.Background(), connect.NewRequest(&struct{}{}))

	require.Equal(t, connect.CodeOf(streamErr), connect.CodeOf(unaryErr))
	require.Equal(t, streamAlerts, alerter.n)

	return unaryErr, alerter
}

func TestErrorInterceptorMapsNoRowsToNotFound(t *testing.T) {
	err, alerter := runThroughErrorInterceptor(t, fmt.Errorf("lookup: %w", pgx.ErrNoRows))

	require.Error(t, err)
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorMapsCanceledWithoutAlert(t *testing.T) {
	err, alerter := runThroughErrorInterceptor(t, context.Canceled)

	require.Error(t, err)
	assert.Equal(t, connect.CodeCanceled, connect.CodeOf(err))
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorUnknownStillInternalAndAlerts(t *testing.T) {
	err, alerter := runThroughErrorInterceptor(t, fmt.Errorf("boom"))

	require.Error(t, err)
	assert.Equal(t, connect.CodeInternal, connect.CodeOf(err))
	assert.NotContains(t, err.Error(), "boom")
	assert.Equal(t, 1, alerter.n)
}

func TestErrorInterceptorPassesThroughExistingCode(t *testing.T) {
	err, alerter := runThroughErrorInterceptor(t, connect.NewError(connect.CodeInvalidArgument, errors.New("bad id")))

	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorKeepsCodeOfGRPCStatusErrors(t *testing.T) {
	err, alerter := runThroughErrorInterceptor(t, status.Error(codes.InvalidArgument, "bad id"))

	require.Error(t, err)
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	var connectErr *connect.Error
	require.ErrorAs(t, err, &connectErr)
	assert.Equal(t, "bad id", connectErr.Message())
	assert.Equal(t, 0, alerter.n)
}

// A status error with details, as an extension built against google.golang.org/grpc returns it,
// reaches the client with every detail, including types the server does not know.
func TestErrorInterceptorKeepsDetailsOfGRPCStatusErrors(t *testing.T) {
	unknownType := &anypb.Any{TypeUrl: "type.googleapis.com/acme.NotRegisteredHere", Value: []byte{0x0a, 0x01, 0x78}}

	collisionDetail, err := anypb.New(&v1contracts.IdempotencyCollisionError{ExistingRunExternalId: "existing-id"})
	require.NoError(t, err)

	// not WithDetails, which would nest the unknown Any inside another one
	statusProto := status.New(codes.AlreadyExists, "idempotency key collision").Proto()
	statusProto.Details = []*anypb.Any{collisionDetail, unknownType}

	st := status.FromProto(statusProto)

	converted, alerter := runThroughErrorInterceptor(t, st.Err())

	var connectErr *connect.Error
	require.ErrorAs(t, converted, &connectErr)

	assert.Equal(t, connect.CodeAlreadyExists, connectErr.Code())
	assert.Equal(t, "idempotency key collision", connectErr.Message())
	assert.Equal(t, 0, alerter.n)

	require.Len(t, connectErr.Details(), 2)

	collision, err := connectErr.Details()[0].Value()
	require.NoError(t, err)
	assert.Equal(t, "existing-id", collision.(*v1contracts.IdempotencyCollisionError).ExistingRunExternalId)

	assert.Equal(t, "acme.NotRegisteredHere", connectErr.Details()[1].Type())
	assert.Equal(t, unknownType.Value, connectErr.Details()[1].Bytes())
}

// The golden text is what grpc-go's status.FromError reports for the same chain built around a
// status error: the whole chain, with the coded error in grpc's rendering.
func TestErrorInterceptorKeepsTheTextOfWrappedCodedErrors(t *testing.T) {
	baseline, _ := status.FromError(fmt.Errorf("could not create trigger opt: %w", status.Error(codes.InvalidArgument, "priority must be between 1 and 3, got 5")))

	coded := connect.NewError(connect.CodeInvalidArgument, errors.New("priority must be between 1 and 3, got 5"))

	detail, err := connect.NewErrorDetail(&v1contracts.IdempotencyCollisionError{ExistingRunExternalId: "existing-id"})
	require.NoError(t, err)

	coded.AddDetail(detail)

	converted, alerter := runThroughErrorInterceptor(t, fmt.Errorf("could not create trigger opt: %w", coded))

	var connectErr *connect.Error
	require.ErrorAs(t, converted, &connectErr)

	assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	assert.Equal(t, baseline.Message(), connectErr.Message())
	assert.Equal(t, "could not create trigger opt: rpc error: code = InvalidArgument desc = priority must be between 1 and 3, got 5", connectErr.Message())
	assert.Len(t, connectErr.Details(), 1)
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorKeepsTheTextOfWrappedGRPCStatusErrors(t *testing.T) {
	wrapped := fmt.Errorf("lookup failed: %w", status.Error(codes.NotFound, "no such worker"))
	baseline, _ := status.FromError(wrapped)

	converted, _ := runThroughErrorInterceptor(t, wrapped)

	var connectErr *connect.Error
	require.ErrorAs(t, converted, &connectErr)

	assert.Equal(t, connect.CodeNotFound, connectErr.Code())
	assert.Equal(t, baseline.Message(), connectErr.Message())
}

func TestErrorInterceptorLeavesDirectCodedErrorsAlone(t *testing.T) {
	coded := connect.NewError(connect.CodeInvalidArgument, errors.New("bad id"))

	converted, _ := runThroughErrorInterceptor(t, coded)

	assert.Same(t, coded, converted)
}

type stubHandlerConn struct {
	connect.StreamingHandlerConn
}

func (stubHandlerConn) Spec() connect.Spec {
	return connect.Spec{}
}
