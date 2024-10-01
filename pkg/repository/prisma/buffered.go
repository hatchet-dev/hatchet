package prisma

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"golang.org/x/sync/errgroup"
)

type ingestBufState int

const (
	started ingestBufState = iota
	initialized
)

// e.g. T is eventOpts and U is *dbsqlc.Event

type IngestBuf[T any, U any] struct {
	outputFunc func(ctx context.Context, items []T) ([]U, error)
	sizeFunc   func(T) int

	state       ingestBufState
	maxCapacity int           // max number of items to hold in buffer before we flush
	flushPeriod time.Duration // max time to hold items in buffer before we flush

	inputChan          chan *inputWrapper[T, U]
	lastFlush          time.Time
	internalArr        []*inputWrapper[T, U]
	sizeOfData         int // size of data in buffer
	maxDataSizeInQueue int // max number of bytes to hold in buffer before we flush

	l      *zerolog.Logger
	lock   sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

type inputWrapper[T any, U any] struct {
	item     T
	doneChan chan<- *flushResponse[U]
}

type IngestBufOpts[T any, U any] struct {
	maxCapacity        int
	flushPeriod        time.Duration
	maxDataSizeInQueue int
	outputFunc         func(ctx context.Context, items []T) ([]U, error)
	sizeFunc           func(T) int
	l                  *zerolog.Logger
}

// NewIngestBuffer creates a new buffer for any type T
func NewIngestBuffer[T any, U any](opts IngestBufOpts[T, U]) *IngestBuf[T, U] {

	inputChannelSize := opts.maxCapacity
	if inputChannelSize < 100 {
		inputChannelSize = 100
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	return &IngestBuf[T, U]{
		state:              initialized,
		maxCapacity:        opts.maxCapacity,
		flushPeriod:        opts.flushPeriod,
		inputChan:          make(chan *inputWrapper[T, U], inputChannelSize),
		lastFlush:          time.Now(),
		internalArr:        make([]*inputWrapper[T, U], 0),
		sizeOfData:         0,
		maxDataSizeInQueue: opts.maxDataSizeInQueue,
		outputFunc:         opts.outputFunc,
		sizeFunc:           opts.sizeFunc,
		l:                  opts.l,
		ctx:                ctx,
		cancel:             cancel,
	}
}

func (b *IngestBuf[T, U]) validate() error {
	if b.maxCapacity <= 0 {
		return fmt.Errorf("max capacity must be greater than 0")
	}
	if b.flushPeriod <= 0 {
		return fmt.Errorf("flush period must be greater than 0")
	}
	if b.maxDataSizeInQueue <= 0 {
		return fmt.Errorf("max data size in queue must be greater than 0")
	}
	if b.outputFunc == nil {
		return fmt.Errorf("bulk create func must be set")
	}
	if b.sizeFunc == nil {
		return fmt.Errorf("size func must be set")
	}
	if b.l == nil {
		return fmt.Errorf("logger must be set")
	}
	return nil
}
func (b *IngestBuf[T, U]) safeFetchSizeOfData() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.sizeOfData
}

func (b *IngestBuf[T, U]) safeIncSizeOfData(size int) {
	b.lock.Lock()
	b.sizeOfData += size
	b.lock.Unlock()
}

func (b *IngestBuf[T, U]) safeDecSizeOfData(size int) {
	b.lock.Lock()
	b.sizeOfData -= size
	b.lock.Unlock()
}

func (b *IngestBuf[T, U]) safeSetLastFlush(t time.Time) {
	b.lock.Lock()
	b.lastFlush = t
	b.lock.Unlock()
}

func (b *IngestBuf[T, U]) safeFetchLastFlush() time.Time {
	b.lock.Lock()
	defer b.lock.Unlock()
	return b.lastFlush
}

func (b *IngestBuf[T, U]) buffWorker() {
	for {
		select {
		case <-b.ctx.Done():
			return
		case e := <-b.inputChan:
			b.internalArr = append(b.internalArr, e)
			b.safeIncSizeOfData(b.calcSizeOfData([]T{e.item}))

			if len(b.internalArr) >= b.maxCapacity {
				go b.flush(b.sliceInternalArray())
			}
			if b.safeFetchSizeOfData() >= b.maxDataSizeInQueue {
				go b.flush(b.sliceInternalArray())
			}

		case <-time.After(time.Until(b.safeFetchLastFlush().Add(b.flushPeriod))):
			if len(b.internalArr) > 0 {
				go b.flush(b.sliceInternalArray())
			} else {
				b.safeSetLastFlush(time.Now())
			}
		}
	}
}

func (b *IngestBuf[T, U]) sliceInternalArray() (items []*inputWrapper[T, U]) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if len(b.internalArr) >= b.maxCapacity {
		items = b.internalArr[:b.maxCapacity]
		b.internalArr = b.internalArr[b.maxCapacity:]
	} else {
		items = b.internalArr
		b.internalArr = nil
	}
	return items
}

type flushResponse[U any] struct {
	result U
	err    error
}

func (b *IngestBuf[T, U]) calcSizeOfData(items []T) int {
	size := 0
	for _, item := range items {
		size += b.sizeFunc(item)
	}
	return size
}

func (b *IngestBuf[T, U]) flush(items []*inputWrapper[T, U]) {
	numItems := len(items)
	b.safeSetLastFlush(time.Now())

	var doneChans []chan<- *flushResponse[U]
	opts := make([]T, numItems)

	for i := 0; i < numItems; i++ {
		opts[i] = items[i].item
		doneChans = append(doneChans, items[i].doneChan)
	}

	b.safeDecSizeOfData(b.calcSizeOfData(opts))

	ctx := context.Background()
	result, err := b.outputFunc(ctx, opts)

	if err != nil {
		for _, doneChan := range doneChans {
			select {
			case doneChan <- &flushResponse[U]{err: err}:
			default:
				b.l.Error().Msgf("could not send error to done chan: %v", err)

			}
		}
		return
	}

	for i, d := range doneChans {
		select {
		case d <- &flushResponse[U]{result: result[i], err: nil}:
		default:
			b.l.Error().Msg("could not send done to done chan")
		}
	}

	b.l.Debug().Msgf("Flushed %d items", numItems)
}

func (b *IngestBuf[T, U]) cleanup() error {
	g := errgroup.Group{}

	for len(b.internalArr) > 0 {
		g.Go(func() error {
			b.flush(b.sliceInternalArray())
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}
	b.cancel()

	return nil
}

func (b *IngestBuf[T, U]) Start() (func() error, error) {
	b.l.Debug().Msg("Starting buffer")

	b.lock.Lock()
	if b.state == started {
		b.lock.Unlock()
		return nil, fmt.Errorf("buffer already started")
	}
	b.state = started
	b.lock.Unlock()

	go b.buffWorker()
	return b.cleanup, nil
}

func (b *IngestBuf[T, U]) BuffItem(item T) (chan *flushResponse[U], error) {
	doneChan := make(chan *flushResponse[U], 1)

	select {
	case b.inputChan <- &inputWrapper[T, U]{
		item:     item,
		doneChan: doneChan,
	}:
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("timeout waiting for buffer")

	case <-b.ctx.Done():
		return nil, fmt.Errorf("buffer is closed")
	}
	return doneChan, nil
}
