package retry

import (
	"context"
	"errors"
	"io"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockDoer struct {
	fn func(req *http.Request, attempt int) (*http.Response, error)
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	return m.fn(req, 0)
}

type countingDoer struct {
	attempts atomic.Int32
	statuses []int
	headers  []http.Header
}

func (c *countingDoer) Do(req *http.Request) (*http.Response, error) {
	attempt := int(c.attempts.Add(1)) - 1
	status := http.StatusOK
	if attempt < len(c.statuses) {
		status = c.statuses[attempt]
	}

	header := make(http.Header)
	if attempt < len(c.headers) && c.headers[attempt] != nil {
		header = c.headers[attempt].Clone()
	}

	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader("ok")),
		Header:     header,
		Request:    req,
	}, nil
}

func newTestRestDoer(inner httpDoer) (*RestDoer, chan time.Duration) {
	slept := make(chan time.Duration, 8)
	sleep := func(ctx context.Context, d time.Duration) error {
		select {
		case slept <- d:
		default:
		}
		return sleepContext(ctx, 0)
	}

	return NewRestDoer(inner, withRestClock(clock{
		now:    time.Now,
		sleep:  sleep,
		jitter: rand.New(rand.NewPCG(1, 1)).Int64N,
	})), slept
}

func TestRestDoerRetriesGatewayStatuses(t *testing.T) {
	t.Parallel()

	for _, status := range []int{http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			t.Parallel()

			inner := &countingDoer{statuses: []int{status, status, http.StatusOK}}
			doer, _ := newTestRestDoer(inner)

			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/path?q=1", nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer token")

			resp, err := doer.Do(req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusOK, resp.StatusCode)
			assert.Equal(t, 3, int(inner.attempts.Load()))

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, "ok", string(body))
			require.NoError(t, resp.Body.Close())
		})
	}
}

func TestRestDoerRetriesTransportErrors(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	inner := &mockDoer{fn: func(req *http.Request, _ int) (*http.Response, error) {
		if attempts.Add(1) < 3 {
			return nil, errors.New("connection reset")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}}

	doer, _ := newTestRestDoer(inner)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestRestDoerDoesNotRetryRequestsOutsideBodylessReads(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		method string
		body   io.ReadCloser
	}{
		{name: "post", method: http.MethodPost, body: nil},
		{name: "delete", method: http.MethodDelete, body: nil},
		{name: "get with body", method: http.MethodGet, body: io.NopCloser(strings.NewReader("x"))},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			inner := &countingDoer{statuses: []int{http.StatusBadGateway}}
			doer, _ := newTestRestDoer(inner)

			req, err := http.NewRequestWithContext(context.Background(), tc.method, "http://example.com", tc.body)
			require.NoError(t, err)

			resp, err := doer.Do(req)
			require.NoError(t, err)
			require.NotNil(t, resp)
			assert.Equal(t, http.StatusBadGateway, resp.StatusCode)
			assert.Equal(t, 1, int(inner.attempts.Load()))
		})
	}
}

func TestRestDoerPreservesRequestClone(t *testing.T) {
	t.Parallel()

	var seenURLs []string
	var seenAuth []string
	inner := &mockDoer{fn: func(req *http.Request, _ int) (*http.Response, error) {
		seenURLs = append(seenURLs, req.URL.String())
		seenAuth = append(seenAuth, req.Header.Get("Authorization"))
		if len(seenURLs) < 2 {
			return &http.Response{
				StatusCode: http.StatusServiceUnavailable,
				Body:       io.NopCloser(strings.NewReader("retry")),
				Header:     make(http.Header),
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}}

	doer, _ := newTestRestDoer(inner)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com/items?x=1", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer abc")

	_, err = doer.Do(req)
	require.NoError(t, err)
	require.Len(t, seenURLs, 2)
	assert.Equal(t, "/items?x=1", seenURLs[0][strings.Index(seenURLs[0], "/items"):])
	assert.Equal(t, []string{"Bearer abc", "Bearer abc"}, seenAuth)
}

func TestRestDoerStopsOnContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	inner := &countingDoer{statuses: []int{http.StatusServiceUnavailable}}
	doer, _ := newTestRestDoer(inner)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	_, err = doer.Do(req)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, int(inner.attempts.Load()))
}

func TestRestDoer429RetryAfterDeltaSeconds(t *testing.T) {
	t.Parallel()

	header := make(http.Header)
	header.Set("Retry-After", "2")
	inner := &countingDoer{
		statuses: []int{http.StatusTooManyRequests, http.StatusOK},
		headers:  []http.Header{header},
	}
	doer, slept := newTestRestDoer(inner)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	select {
	case delay := <-slept:
		assert.Equal(t, 2*time.Second, delay)
	case <-time.After(time.Second):
		t.Fatal("expected retry sleep delay")
	}
}

func TestRestDoer429InvalidRetryAfterStillRetries(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32
	inner := &mockDoer{fn: func(req *http.Request, _ int) (*http.Response, error) {
		attempt := int(attempts.Add(1))
		if attempt < 2 {
			header := make(http.Header)
			header.Set("Retry-After", "invalid")
			return &http.Response{
				StatusCode: http.StatusTooManyRequests,
				Body:       io.NopCloser(strings.NewReader("retry")),
				Header:     header,
				Request:    req,
			}, nil
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
			Request:    req,
		}, nil
	}}

	doer, _ := newTestRestDoer(inner)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), attempts.Load())
}

func TestRestDoerCallerDeadlineOverridesHeaderTimeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(20 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	doer := NewRestDoer(server.Client(), withHeaderTimeout(time.Millisecond))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.NoError(t, resp.Body.Close())
}

func TestRestDoerFinalBodyReadablePastHeaderTimeout(t *testing.T) {
	t.Parallel()

	payload := []byte("slow-body")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		time.Sleep(150 * time.Millisecond)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	doer := NewRestDoer(server.Client(), withHeaderTimeout(50*time.Millisecond))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, payload, body)
	require.NoError(t, resp.Body.Close())
}

func TestRestDoerDoesNotRetry404(t *testing.T) {
	t.Parallel()

	inner := &countingDoer{statuses: []int{http.StatusNotFound}}
	doer, _ := newTestRestDoer(inner)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	require.NoError(t, err)

	resp, err := doer.Do(req)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	assert.Equal(t, 1, int(inner.attempts.Load()))
}
