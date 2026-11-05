package prometheus

import (
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func freeAddress(t *testing.T) string {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	return addr
}

func metricsStatus(addr string) (int, error) {
	client := http.Client{Timeout: time.Second}

	resp, err := client.Get("http://" + addr + "/metrics")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	return resp.StatusCode, nil
}

func TestStartServerSharesAddressUntilLastStop(t *testing.T) {
	l := zerolog.Nop()
	addr := freeAddress(t)

	stopFirst := StartServer(&l, addr, "/metrics")

	require.Eventually(t, func() bool {
		status, err := metricsStatus(addr)
		return err == nil && status == http.StatusOK
	}, 5*time.Second, 20*time.Millisecond)

	first := servers[addr]
	stopSecond := StartServer(&l, addr, "/metrics")

	// checked on the shared state: a second server's bind races the first one's shutdown, so the
	// HTTP checks alone can pass without sharing
	require.Same(t, first, servers[addr])
	require.Equal(t, 2, servers[addr].refs)

	require.NoError(t, stopFirst())
	require.NoError(t, stopFirst(), "a second stop by the same caller must not release another reference")

	status, err := metricsStatus(addr)
	require.NoError(t, err, "the server must keep serving while another caller still holds it")
	require.Equal(t, http.StatusOK, status)

	require.NoError(t, stopSecond())

	_, err = metricsStatus(addr)
	require.Error(t, err, "the server must stop after the last caller stops it")
}
