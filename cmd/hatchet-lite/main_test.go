package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReverseProxyForwardsClientIP(t *testing.T) {
	var apiXFF, apiXRealIP, feXFF string

	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiXFF = r.Header.Get("X-Forwarded-For")
		apiXRealIP = r.Header.Get("X-Real-Ip")
	}))
	defer api.Close()

	fe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		feXFF = r.Header.Get("X-Forwarded-For")
	}))
	defer fe.Close()

	apiURL, err := url.Parse(api.URL)
	require.NoError(t, err)

	feURL, err := url.Parse(fe.URL)
	require.NoError(t, err)

	proxy := newReverseProxy(apiURL, feURL)

	do := func(path string, header http.Header) {
		req := httptest.NewRequest(http.MethodPost, path, nil)
		req.RemoteAddr = "203.0.113.7:54321"
		req.Header = header

		proxy.ServeHTTP(httptest.NewRecorder(), req)
	}

	do("/api/v1/users/login", http.Header{})
	require.Equal(t, "203.0.113.7", apiXFF)

	do("/api/v1/users/login", http.Header{"X-Forwarded-For": {"1.1.1.1"}, "X-Real-Ip": {"1.1.1.1"}})
	require.Equal(t, "203.0.113.7", apiXFF, "client-supplied X-Forwarded-For is dropped")
	require.Empty(t, apiXRealIP, "client-supplied X-Real-IP is dropped")

	do("/", http.Header{})
	require.Equal(t, "203.0.113.7", feXFF)
}
