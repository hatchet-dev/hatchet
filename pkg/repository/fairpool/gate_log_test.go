//go:build !e2e && !load && !rampup && !integration

package fairpool

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func noopOpen() error { return nil }

func newLoggedGate(t *testing.T, limit int64, maxWait time.Duration) (*gate, *bytes.Buffer) {
	t.Helper()

	buf := &bytes.Buffer{}
	l := zerolog.New(buf)

	return newGate(limit, maxWait, "test", &l), buf
}

func logLines(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var lines []map[string]any

	for _, raw := range bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte("\n")) {
		if len(raw) == 0 {
			continue
		}

		line := map[string]any{}
		require.NoError(t, json.Unmarshal(raw, &line))
		lines = append(lines, line)
	}

	return lines
}

// fillAndReject holds key's only slot, then rejects n further checkouts.
func fillAndReject(t *testing.T, g *gate, key string, n int) {
	t.Helper()

	hold, err := g.pin(context.Background(), key, uuid.Nil, noopOpen)
	require.NoError(t, err)
	t.Cleanup(hold.release)

	for i := 0; i < n; i++ {
		_, err := g.pin(context.Background(), key, uuid.Nil, noopOpen)
		require.ErrorAs(t, err, new(*LimitError))
	}
}

func TestRejectionLogCountsSuppressedRejections(t *testing.T) {
	g, buf := newLoggedGate(t, 1, time.Millisecond)

	fillAndReject(t, g, "tenant-a", 5)

	// the first rejection logs at once; the other four fall inside the interval and are counted
	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	require.EqualValues(t, 1, lines[0]["rejected"])

	g.mu.Lock()
	require.EqualValues(t, 4, g.tenants["tenant-a"].rejected)
	g.tenants["tenant-a"].lastRejectLog = time.Now().Add(-2 * logInterval)
	g.mu.Unlock()

	_, err := g.pin(context.Background(), "tenant-a", uuid.Nil, noopOpen)
	require.ErrorAs(t, err, new(*LimitError))

	lines = logLines(t, buf)
	require.Len(t, lines, 2)

	line := lines[1]
	require.Equal(t, "warn", line["level"])
	require.Equal(t, "fairpool: tenant checkout rejected at cap", line["message"])
	require.EqualValues(t, 5, line["rejected"])
	require.Equal(t, "tenant-a", line["tenant_id"])
	require.Equal(t, "test", line["pool"])
	require.Equal(t, "tenant", line["bucket"])
	require.EqualValues(t, 1, line["limit"])
	require.Contains(t, line, "waited")

	g.mu.Lock()
	require.Zero(t, g.tenants["tenant-a"].rejected)
	g.mu.Unlock()
}

func TestRejectionLogSharedBucket(t *testing.T) {
	g, buf := newLoggedGate(t, 1, time.Millisecond)

	fillAndReject(t, g, sharedKey, 1)

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	require.Equal(t, "fairpool: shared work checkout rejected at cap", lines[0]["message"])
	require.Equal(t, "shared", lines[0]["tenant_id"])
	require.Equal(t, "shared", lines[0]["bucket"])
}

func TestRejectionLogBurstBoundsKeysAndCarriesCounts(t *testing.T) {
	const burst = 3

	g, buf := newLoggedGate(t, 1, time.Millisecond)

	sample := func(burst uint32) zerolog.Logger {
		return zerolog.New(buf).Sample(&zerolog.BurstSampler{Burst: burst, Period: time.Hour})
	}

	g.rejectLog = sample(burst)

	keys := []string{"k0", "k1", "k2", "k3", "k4"}

	for _, key := range keys {
		fillAndReject(t, g, key, 1)
	}

	// only the burst's worth of keys logged
	logged := map[string]bool{}
	for _, line := range logLines(t, buf) {
		logged[line["tenant_id"].(string)] = true
	}

	require.Len(t, logged, burst)

	var dropped string

	for _, key := range keys {
		if !logged[key] {
			dropped = key
			break
		}
	}

	require.NotEmpty(t, dropped)

	// a dropped key keeps accumulating, and none of its rejections are lost
	_, err := g.pin(context.Background(), dropped, uuid.Nil, noopOpen)
	require.ErrorAs(t, err, new(*LimitError))
	require.Len(t, logLines(t, buf), burst)

	// a fresh period lets the next rejection through with the whole count
	g.rejectLog = sample(10)

	_, err = g.pin(context.Background(), dropped, uuid.Nil, noopOpen)
	require.ErrorAs(t, err, new(*LimitError))

	lines := logLines(t, buf)
	require.Len(t, lines, burst+1)
	require.Equal(t, dropped, lines[burst]["tenant_id"])
	require.EqualValues(t, 3, lines[burst]["rejected"], fmt.Sprint(lines[burst]))
}

func TestWaitLogIsInfoAndSeparateFromRejections(t *testing.T) {
	g, buf := newLoggedGate(t, 1, time.Second)

	hold, err := g.pin(context.Background(), "tenant-a", uuid.Nil, noopOpen)
	require.NoError(t, err)

	go func() {
		time.Sleep(20 * time.Millisecond)
		hold.release()
	}()

	waitedHold, err := g.pin(context.Background(), "tenant-a", uuid.Nil, noopOpen)
	require.NoError(t, err)
	t.Cleanup(waitedHold.release)

	lines := logLines(t, buf)
	require.Len(t, lines, 1)
	require.Equal(t, "info", lines[0]["level"])
	require.Equal(t, "fairpool: tenant waited for a connection slot", lines[0]["message"])
	require.Equal(t, "tenant-a", lines[0]["tenant_id"])
	require.Equal(t, "tenant", lines[0]["bucket"])
	require.NotContains(t, lines[0], "rejected")

	// a wait line does not hold back the first rejection line
	g.maxWait = time.Millisecond

	_, err = g.pin(context.Background(), "tenant-a", uuid.Nil, noopOpen)
	require.ErrorAs(t, err, new(*LimitError))

	lines = logLines(t, buf)
	require.Len(t, lines, 2)
	require.Equal(t, "fairpool: tenant checkout rejected at cap", lines[1]["message"])
}
