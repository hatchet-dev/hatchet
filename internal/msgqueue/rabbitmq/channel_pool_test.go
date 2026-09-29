package rabbitmq

import (
	"context"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"

	"github.com/jackc/puddle/v2"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/rs/zerolog"
)

func TestRedactURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "username and password",
			in:   "amqp://user:s3cret@rabbitmq-blue:5672/",
			want: "amqp://user:xxxxx@rabbitmq-blue:5672/",
		},
		{
			name: "username only",
			in:   "amqp://user@rabbitmq:5672/vhost",
			want: "amqp://user@rabbitmq:5672/vhost",
		},
		{
			name: "no credentials",
			in:   "amqp://rabbitmq:5672/",
			want: "amqp://rabbitmq:5672/",
		},
		{
			name: "unparseable",
			in:   "amqp://user:pass@rabbit:5672/\x7f%zz",
			want: "<unparseable url>",
		},
		{
			// A scheme-less string parses without a host, and Redacted would
			// return the credentials verbatim, so it must be masked entirely.
			name: "scheme-less with credentials",
			in:   "user:pass@rabbit:5672/vhost",
			want: "<unparseable url>",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := redactURL(tt.in)

			if got != tt.want {
				t.Errorf("redactURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestChannelPoolReconnectDropsStaleChannels models a publisher connection that
// dies with a full pool of idle channels and is then redialed: every idle
// channel belongs to the dead connection, and puddle hands idle resources out
// before constructing new ones, so a publisher that gives up after a few closed
// channels would drop its batch against a healthy broker. After reconnect the
// pool must hand out a channel constructed on the new connection.
func TestChannelPoolReconnectDropsStaleChannels(t *testing.T) {
	const maxChannels = 20

	l := zerolog.Nop()
	var dials, constructed atomic.Int32

	p := &channelPool{
		l:   &l,
		url: "amqp://test",
		dial: func(string) (*amqp.Connection, error) {
			dials.Add(1)
			// never used for I/O in this test: the pool only stores it
			return &amqp.Connection{}, nil
		},
	}

	pool, err := puddle.NewPool(&puddle.Config[*amqp.Channel]{
		Constructor: func(context.Context) (*amqp.Channel, error) {
			constructed.Add(1)
			return &amqp.Channel{}, nil
		},
		Destructor: func(*amqp.Channel) {},
		MaxSize:    maxChannels,
	})
	if err != nil {
		t.Fatal(err)
	}
	p.Pool = pool
	defer p.Close()

	if err := p.newConnection(); err != nil {
		t.Fatal(err)
	}

	// every resource acquired below is tracked and released on exit, registered
	// before the first acquisition so that a failure at any point cannot leave
	// Close waiting on an acquired resource
	ctx := context.Background()
	open := map[*puddle.Resource[*amqp.Channel]]struct{}{}
	defer func() {
		for res := range open {
			res.Release()
		}
	}()
	acquire := func() *puddle.Resource[*amqp.Channel] {
		res, err := p.Acquire(ctx)
		if err != nil {
			t.Fatal(err)
		}
		open[res] = struct{}{}
		return res
	}
	release := func(res *puddle.Resource[*amqp.Channel]) {
		res.Release()
		delete(open, res)
	}

	// fill the pool with channels opened on the first connection and park them
	// idle, keeping one acquired across the reconnect as an in-flight publish would
	held := make([]*puddle.Resource[*amqp.Channel], 0, maxChannels)
	for range maxChannels {
		held = append(held, acquire())
	}
	inFlight := held[0]
	for _, res := range held[1:] {
		release(res)
	}

	if got := constructed.Load(); got != maxChannels {
		t.Fatalf("expected %d channels constructed on the first connection, got %d", maxChannels, got)
	}

	if err := p.reconnect(); err != nil {
		t.Fatal(err)
	}

	if got := dials.Load(); got != 2 {
		t.Fatalf("expected the connection to be redialed once, got %d dials", got)
	}

	if idle := p.Stat().IdleResources(); idle != 0 {
		t.Fatalf("expected no idle channels from the dead connection after reconnect, got %d", idle)
	}

	// the next acquisition must construct on the new connection rather than hand out a dead channel
	acquire()
	if got := constructed.Load(); got != maxChannels+1 {
		t.Fatalf("expected a freshly constructed channel after reconnect, constructor calls went %d -> %d", maxChannels, got)
	}

	// the channel that was in flight during the reconnect must not return to the idle set
	release(inFlight)
	if total := p.Stat().TotalResources(); total != 1 {
		t.Fatalf("expected only the post-reconnect channel to remain in the pool, got %d", total)
	}
}

// markConnectionClosed flips the pinned client's private closed flag so that
// IsClosed reports true without any broker or socket. The field is an
// atomic.Bool named "closed" on amqp091-go v1.15.0.
func markConnectionClosed(t *testing.T, conn *amqp.Connection) {
	t.Helper()

	field := reflect.ValueOf(conn).Elem().FieldByName("closed")
	if !field.IsValid() {
		t.Fatal("amqp.Connection has no closed field; update markConnectionClosed for this client version")
	}

	(*atomic.Bool)(unsafe.Pointer(field.UnsafeAddr())).Store(true) // #nosec G103 -- test-only access to a private flag
}

// TestChannelPoolReconnectsOnlyAfterPoolIsReady covers the startup ordering: the
// initial connection dies right away, so the watcher's first tick redials and
// resets the pool. The pool must already be assigned by then; with the watcher
// started before the pool existed, Reset dereferenced a nil pool and the process
// panicked during startup.
func TestChannelPoolReconnectsOnlyAfterPoolIsReady(t *testing.T) {
	orig := reconnectCheckInterval
	reconnectCheckInterval = time.Millisecond
	defer func() { reconnectCheckInterval = orig }()

	var dials atomic.Int32
	dial := func(string) (*amqp.Connection, error) {
		conn := &amqp.Connection{}
		if dials.Add(1) == 1 {
			markConnectionClosed(t, conn)
		}
		return conn, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	l := zerolog.Nop()
	p, err := newChannelPoolWithDial(ctx, &l, "amqp://test", 4, channelPoolQueueDurable, channelPoolRolePub, dial)
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()

	deadline := time.After(2 * time.Second)
	for dials.Load() < 2 {
		select {
		case <-deadline:
			t.Fatalf("watcher did not redial the closed connection, dials=%d", dials.Load())
		case <-time.After(time.Millisecond):
		}
	}

	if !p.hasActiveConnection() {
		t.Fatal("expected the replacement connection to be active")
	}

	if total := p.Stat().TotalResources(); total != 0 {
		t.Fatalf("expected an empty pool after the startup reset, got %d resources", total)
	}
}
