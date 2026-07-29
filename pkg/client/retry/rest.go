package retry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

var errInvalidRestMaxAttempts = errors.New("rest max attempts must be at least 1")

func validateRestMaxAttempts(maxAttempts int) error {
	if maxAttempts < 1 {
		return fmt.Errorf("%w: got %d", errInvalidRestMaxAttempts, maxAttempts)
	}
	return nil
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// RestDoer retries eligible bodyless GET and HEAD requests.
type RestDoer struct {
	inner              httpDoer
	headerTimeoutInner httpDoer
	clock              clock
	maxAttempts        int
}

type restDoerOption func(*RestDoer)

func withRestClock(c clock) restDoerOption {
	return func(d *RestDoer) {
		d.clock = c
	}
}

func withHeaderTimeout(timeout time.Duration) restDoerOption {
	return func(d *RestDoer) {
		d.headerTimeoutInner = withResponseHeaderTimeout(d.inner, timeout)
	}
}

// NewRestDoer wraps inner with REST read retry behavior.
func NewRestDoer(inner httpDoer, opts ...restDoerOption) (*RestDoer, error) {
	if err := validateRestMaxAttempts(restMaxAttempts); err != nil {
		return nil, err
	}

	if inner == nil {
		inner = &http.Client{}
	}

	d := &RestDoer{
		inner:              inner,
		headerTimeoutInner: withResponseHeaderTimeout(inner, restPerAttemptTimeout),
		clock:              defaultClock(),
		maxAttempts:        restMaxAttempts,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d, nil
}

func (d *RestDoer) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return d.inner.Do(req)
	}

	if err := req.Context().Err(); err != nil {
		return nil, err
	}

	if !isRequestEligibleForRetry(req) {
		return d.inner.Do(req)
	}

	ctx := req.Context()
	original := req.Clone(ctx)

	var lastTransportErr error

	for attempt := range d.maxAttempts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		attemptReq := original.Clone(ctx)
		resp, err := d.doAttempt(ctx, attemptReq)
		if err != nil {
			lastTransportErr = err

			if attempt == d.maxAttempts-1 || ctx.Err() != nil {
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}
				return nil, err
			}

			delay := d.restBackoffDelay(attempt, nil)
			if err := d.clock.sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if !isResponseStatusCodeEligibleForRetry(resp.StatusCode) || attempt == d.maxAttempts-1 {
			return resp, nil
		}

		discardResponse(resp)

		delay := d.restBackoffDelay(attempt, resp)
		if err := d.clock.sleep(ctx, delay); err != nil {
			return nil, err
		}
	}

	if lastTransportErr != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, lastTransportErr
	}

	return nil, ctx.Err()
}

func (d *RestDoer) doAttempt(ctx context.Context, req *http.Request) (*http.Response, error) {
	if _, hasDeadline := ctx.Deadline(); hasDeadline {
		return d.inner.Do(req)
	}

	return d.headerTimeoutInner.Do(req)
}

func isRequestEligibleForRetry(req *http.Request) bool {
	if req == nil {
		return false
	}

	if req.Method != http.MethodGet && req.Method != http.MethodHead {
		return false
	}

	if req.Body != nil && req.Body != http.NoBody {
		return false
	}

	return true
}

func isResponseStatusCodeEligibleForRetry(statusCode int) bool {
	switch statusCode {
	case http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout,
		http.StatusTooManyRequests:
		return true
	default:
		return false
	}
}

func (d *RestDoer) restBackoffDelay(attempt int, resp *http.Response) time.Duration {
	if resp != nil && resp.StatusCode == http.StatusTooManyRequests {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), d.clock.now(), restMaxRetryAfter); ok {
			return delay
		}
	}

	return fullJitterDelay(attempt, restBaseDelay, restBackoffFactor, restMaxDelay, d.clock.jitter)
}

func discardResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}

	_, _ = io.CopyN(io.Discard, resp.Body, restDiscardDrainLimitBytes)
	_ = resp.Body.Close()
}

func withResponseHeaderTimeout(inner httpDoer, timeout time.Duration) httpDoer {
	client, ok := inner.(*http.Client)
	if !ok {
		return inner
	}

	return clientWithResponseHeaderTimeout(client, timeout)
}

func clientWithResponseHeaderTimeout(client *http.Client, timeout time.Duration) *http.Client {
	if client == nil {
		client = http.DefaultClient
	}

	cloned := *client
	transport := cloned.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}

	t, ok := transport.(*http.Transport)
	if !ok {
		return client
	}

	tc := t.Clone()
	tc.ResponseHeaderTimeout = timeout
	cloned.Transport = tc
	return &cloned
}
