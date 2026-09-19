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

type stubHandlerConn struct {
	connect.StreamingHandlerConn
}

func (stubHandlerConn) Spec() connect.Spec {
	return connect.Spec{}
}
