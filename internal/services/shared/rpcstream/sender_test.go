package rpcstream

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingStream struct {
	entered chan struct{}
	release chan struct{}
	sent    []int
}

func (b *blockingStream) Send(msg *int) error {
	b.entered <- struct{}{}
	<-b.release

	b.sent = append(b.sent, *msg)

	return nil
}

func TestSenderRejectsSendsAfterClose(t *testing.T) {
	stream := &blockingStream{entered: make(chan struct{}, 1), release: make(chan struct{})}
	close(stream.release)

	sender := NewSender[int](context.Background(), stream)

	one, two := 1, 2
	require.NoError(t, sender.Send(&one))

	sender.Close()

	assert.ErrorIs(t, sender.Send(&two), ErrClosed)
	assert.Equal(t, []int{1}, stream.sent)
}

// Close must not return while a Send is still writing: the handler returns right after Close,
// and the HTTP/2 server panics on a write that outlives its handler.
func TestSenderCloseWaitsForInFlightSend(t *testing.T) {
	stream := &blockingStream{entered: make(chan struct{}), release: make(chan struct{})}
	sender := NewSender[int](context.Background(), stream)

	var wg sync.WaitGroup

	wg.Go(func() {
		one := 1
		assert.NoError(t, sender.Send(&one))
	})

	<-stream.entered

	closed := make(chan struct{})

	go func() {
		sender.Close()
		close(closed)
	}()

	select {
	case <-closed:
		t.Fatal("Close returned while a Send was in flight")
	case <-time.After(50 * time.Millisecond):
	}

	close(stream.release)
	wg.Wait()

	select {
	case <-closed:
	case <-time.After(time.Second):
		t.Fatal("Close did not return after the Send finished")
	}
}
