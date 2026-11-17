package loader

import (
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgproto3"
	"github.com/stretchr/testify/require"
)

func TestNewPgxPoolConfig_ConnectTimeoutOverridesURL(t *testing.T) {
	cfg, err := NewPgxPoolConfig("postgres://user@127.0.0.1:5432/db?connect_timeout=120", PgxPoolOpts{})
	require.NoError(t, err)

	require.Equal(t, defaultPgxConnectTimeout, cfg.ConnConfig.ConnectTimeout)
}

func TestBoundPgxConnect_SlowServerVsSilentServer(t *testing.T) {
	const dialTimeout, connectTimeout = 200 * time.Millisecond, time.Second

	tests := []struct {
		name    string
		delay   time.Duration
		wantErr bool
	}{
		{name: "server slower than the dial timeout still connects", delay: 500 * time.Millisecond},
		{name: "server that never answers stops at the connect timeout", delay: time.Hour, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := pgx.ParseConfig(fakePostgres(t, tc.delay))
			require.NoError(t, err)

			boundPgxConnect(cfg, dialTimeout, connectTimeout)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			start := time.Now()
			conn, err := pgx.ConnectConfig(ctx, cfg)
			elapsed := time.Since(start)

			if tc.wantErr {
				require.Error(t, err)
				require.Less(t, elapsed, connectTimeout+500*time.Millisecond)
				return
			}

			require.NoError(t, err)
			require.NoError(t, conn.Close(ctx))
		})
	}
}

func TestPgxAfterConnect_BoundedByConnectTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// the fake server completes startup but never answers a query, like a stalled backend
	conn, err := pgx.Connect(ctx, fakePostgres(t, 0))
	require.NoError(t, err)

	start := time.Now()
	err = pgxAfterConnect(time.Minute, 100*time.Millisecond)(ctx, conn)

	require.Error(t, err)
	require.Less(t, time.Since(start), time.Second)
}

// fakePostgres accepts connections on localhost, completes a trust-auth startup after delay and
// then reads queries without answering them.
func fakePostgres(t *testing.T, delay time.Duration) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	done := make(chan struct{})
	t.Cleanup(func() {
		close(done)
		_ = ln.Close()
	})

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}

			go func() {
				defer conn.Close()

				backend := pgproto3.NewBackend(conn, conn)
				if _, err := backend.ReceiveStartupMessage(); err != nil {
					return
				}

				select {
				case <-time.After(delay):
				case <-done:
					return
				}

				backend.Send(&pgproto3.AuthenticationOk{})
				backend.Send(&pgproto3.ReadyForQuery{TxStatus: 'I'})
				if err := backend.Flush(); err != nil {
					return
				}

				_, _ = io.Copy(io.Discard, conn)
			}()
		}
	}()

	return "postgres://user@" + ln.Addr().String() + "/db?sslmode=disable"
}
