//go:build !e2e && !load && !rampup && !integration

package loaderutils

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/pkg/config/shared"
)

func TestParseTLSMinVersion(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    uint16
		wantErr bool
	}{
		{name: "empty defaults to TLS 1.3", input: "", want: tls.VersionTLS13},
		{name: "1.3", input: "1.3", want: tls.VersionTLS13},
		{name: "tls1.3", input: "tls1.3", want: tls.VersionTLS13},
		{name: "TLS1.3 uppercase", input: "TLS1.3", want: tls.VersionTLS13},
		{name: "tls_1.3", input: "tls_1.3", want: tls.VersionTLS13},
		{name: "tls13", input: "tls13", want: tls.VersionTLS13},
		{name: "1.2", input: "1.2", want: tls.VersionTLS12},
		{name: "tls1.2", input: "tls1.2", want: tls.VersionTLS12},
		{name: "TLS1.2 uppercase", input: "TLS1.2", want: tls.VersionTLS12},
		{name: "tls_1.2", input: "tls_1.2", want: tls.VersionTLS12},
		{name: "tls12", input: "tls12", want: tls.VersionTLS12},
		{name: "leading/trailing whitespace", input: "  1.2  ", want: tls.VersionTLS12},
		{name: "1.1 rejected", input: "1.1", wantErr: true},
		{name: "1.0 rejected", input: "1.0", wantErr: true},
		{name: "arbitrary string rejected", input: "latest", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTLSMinVersion(tc.input)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unsupported TLS minimum version")
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestLoadBaseTLSConfig_DefaultMinVersion(t *testing.T) {
	cfg := &shared.TLSConfigFile{}
	res, _, err := LoadBaseTLSConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, uint16(tls.VersionTLS13), res.MinVersion)
}

func TestLoadBaseTLSConfig_ConfiguredMinVersion(t *testing.T) {
	cfg := &shared.TLSConfigFile{TLSMinVersion: "1.2"}
	res, _, err := LoadBaseTLSConfig(cfg)
	require.NoError(t, err)
	assert.Equal(t, uint16(tls.VersionTLS12), res.MinVersion)
}

func TestLoadBaseTLSConfig_InvalidMinVersion(t *testing.T) {
	cfg := &shared.TLSConfigFile{TLSMinVersion: "1.0"}
	_, _, err := LoadBaseTLSConfig(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported TLS minimum version")
}

func TestTLS12Handshake(t *testing.T) {
	cert := generateSelfSignedCert(t)

	serverCfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MaxVersion:   tls.VersionTLS12,
		MinVersion:   tls.VersionTLS12,
	}

	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	require.NoError(t, err)
	defer ln.Close()

	addr := ln.Addr().String()

	t.Run("TLS 1.3 client fails against TLS 1.2 server", func(t *testing.T) {
		serverDone := acceptAndClose(ln)

		conn, err := tls.Dial("tcp", addr, &tls.Config{
			MinVersion:         tls.VersionTLS13,
			InsecureSkipVerify: true, //nolint:gosec // self-signed test cert
		})
		if err == nil {
			conn.Close()
			t.Fatal("expected handshake failure with TLS 1.3 minimum against TLS 1.2 server")
		}

		<-serverDone
	})

	t.Run("TLS 1.2 client succeeds against TLS 1.2 server", func(t *testing.T) {
		serverDone := acceptAndClose(ln)

		conn, err := tls.Dial("tcp", addr, &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: true, //nolint:gosec // self-signed test cert
		})
		require.NoError(t, err)
		conn.Close()

		<-serverDone
	})
}

// acceptAndClose accepts one TLS connection and closes done when the server
// goroutine exits, allowing the test to wait for handshake cleanup.
func acceptAndClose(ln net.Listener) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		if tlsConn, ok := conn.(*tls.Conn); ok {
			_ = tlsConn.Handshake()
		}
		conn.Close()
	}()
	return done
}

func generateSelfSignedCert(t *testing.T) tls.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{Organization: []string{"Test"}},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
	}

	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	return tls.Certificate{
		Certificate: [][]byte{derBytes},
		PrivateKey:  key,
	}
}
