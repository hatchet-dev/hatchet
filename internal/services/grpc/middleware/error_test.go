package middleware

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type countingAlerter struct {
	n int
}

func (c *countingAlerter) SendAlert(ctx context.Context, err error, data map[string]interface{}) {
	c.n++
}

func unaryInterceptor() (*ErrorInterceptor, *countingAlerter) {
	alerter := &countingAlerter{}
	l := zerolog.Nop()
	return NewErrorInterceptor(alerter, &l), alerter
}

func TestErrorInterceptorMapsNoRowsToNotFound(t *testing.T) {
	e, alerter := unaryInterceptor()
	interceptor := e.ErrorUnaryServerInterceptor()

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		return nil, fmt.Errorf("lookup: %w", pgx.ErrNoRows)
	})

	require.Error(t, err)
	st, ok := status.FromError(err)
	require.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorMapsCanceledWithoutAlert(t *testing.T) {
	e, alerter := unaryInterceptor()
	interceptor := e.ErrorUnaryServerInterceptor()

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		return nil, context.Canceled
	})

	require.Error(t, err)
	assert.Equal(t, codes.Canceled, status.Code(err))
	assert.Equal(t, 0, alerter.n)
}

func TestErrorInterceptorUnknownStillInternalAndAlerts(t *testing.T) {
	e, alerter := unaryInterceptor()
	interceptor := e.ErrorUnaryServerInterceptor()

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		return nil, fmt.Errorf("boom")
	})

	require.Error(t, err)
	assert.Equal(t, codes.Internal, status.Code(err))
	assert.Equal(t, 1, alerter.n)
}

func TestErrorInterceptorPassesThroughExistingStatus(t *testing.T) {
	e, alerter := unaryInterceptor()
	interceptor := e.ErrorUnaryServerInterceptor()

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test"}, func(ctx context.Context, req any) (any, error) {
		return nil, status.Error(codes.InvalidArgument, "bad id")
	})

	require.Error(t, err)
	assert.Equal(t, codes.InvalidArgument, status.Code(err))
	assert.Equal(t, 0, alerter.n)
}

func TestErrorStreamInterceptorMapsNoRowsToNotFound(t *testing.T) {
	e, alerter := unaryInterceptor()
	interceptor := e.ErrorStreamServerInterceptor()

	err := interceptor(nil, &stubServerStream{ctx: context.Background()}, &grpc.StreamServerInfo{FullMethod: "/test"}, func(srv any, stream grpc.ServerStream) error {
		return pgx.ErrNoRows
	})

	require.Error(t, err)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Equal(t, 0, alerter.n)
}

type stubServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *stubServerStream) Context() context.Context {
	return s.ctx
}
