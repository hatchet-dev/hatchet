package buffer

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
	finished
)

func (s ingestBufState) String() string {
	return [...]string{"started", "initialized", "finished"}[s]
}

// e.g. T is eventOpts and U is *dbsqlc.Event

type IngestBuf[T any, U any] struct {
	name       string // a human readable name for the buffer
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
	doneChan chan<- *FlushResponse[U]
}

type IngestBufOpts[T any, U any] struct {
	Name               string                                            `validate:"required"`
	MaxCapacity        int                                               `validate:"required,gt=0"`
	FlushPeriod        time.Duration                                     `validate:"required,gt=0"`
	MaxDataSizeInQueue int                                               `validate:"required,gt=0"`
	OutputFunc         func(ctx context.Context, items []T) ([]U, error) `validate:"required"`
	SizeFunc           func(T) int                                       `validate:"required"`
	L                  *zerolog.Logger                                   `validate:"required"`
}

// NewIngestBuffer creates a new buffer for any type T
func NewIngestBuffer[T any, U any](opts IngestBufOpts[T, U]) *IngestBuf[T, U] {

	inputChannelSize := opts.MaxCapacity
	if inputChannelSize < 100 {
		inputChannelSize = 100
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	logger := opts.L.With().Str("buffer", opts.Name).Logger()

	return &IngestBuf[T, U]{
		name:               opts.Name,
		state:              initialized,
		maxCapacity:        opts.MaxCapacity,
		flushPeriod:        opts.FlushPeriod,
		inputChan:          make(chan *inputWrapper[T, U], inputChannelSize),
		lastFlush:          time.Now(),
		internalArr:        make([]*inputWrapper[T, U], 0),
		sizeOfData:         0,
		maxDataSizeInQueue: opts.MaxDataSizeInQueue,
		outputFunc:         opts.OutputFunc,
		sizeFunc:           opts.SizeFunc,
		l:                  &logger,
		ctx:                ctx,
		cancel:             cancel,
	}
}

func (b *IngestBuf[T, U]) safeAppendInternalArray(e *inputWrapper[T, U]) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.internalArr = append(b.internalArr, e)
}

func (b *IngestBuf[T, U]) safeFetchSizeOfData() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return b.sizeOfData
}

func (b *IngestBuf[T, U]) safeIncSizeOfData(size int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sizeOfData += size

}

func (b *IngestBuf[T, U]) safeDecSizeOfData(size int) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.sizeOfData -= size

}

func (b *IngestBuf[T, U]) safeSetLastFlush(t time.Time) {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.lastFlush = t

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
			b.safeAppendInternalArray(e)
			b.safeIncSizeOfData(b.calcSizeOfData([]T{e.item}))

			// if last flush time + flush period is in the past, flush
			if time.Now().After(b.safeFetchLastFlush().Add(b.flushPeriod)) {
				b.flush(b.sliceInternalArray())
			} else if b.safeCheckSizeOfBuffer() >= b.maxCapacity || b.safeFetchSizeOfData() >= b.maxDataSizeInQueue {
				b.flush(b.sliceInternalArray())
			}
		case <-time.After(time.Until(b.safeFetchLastFlush().Add(b.flushPeriod))):

			b.flush(b.sliceInternalArray())

		}
	}
}

func (b *IngestBuf[T, U]) sliceInternalArray() (items []*inputWrapper[T, U]) {

	if b.safeCheckSizeOfBuffer() >= b.maxCapacity {
		b.lock.Lock()
		defer b.lock.Unlock()
		items = b.internalArr[:b.maxCapacity]
		b.internalArr = b.internalArr[b.maxCapacity:]
	} else {
		b.lock.Lock()
		defer b.lock.Unlock()
		items = b.internalArr
		b.internalArr = nil
	}
	return items
}

type FlushResponse[U any] struct {
	Result U
	Err    error
}

func (b *IngestBuf[T, U]) calcSizeOfData(items []T) int {
	size := 0
	for _, item := range items {
		size += b.sizeFunc(item)
	}
	return size
}

func (b *IngestBuf[T, U]) safeCheckSizeOfBuffer() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	return len(b.internalArr)
}

func (b *IngestBuf[T, U]) flush(items []*inputWrapper[T, U]) {
	numItems := len(items)
	b.safeSetLastFlush(time.Now())

	if numItems == 0 {
		// nothing to flush
		return
	}

	var doneChans []chan<- *FlushResponse[U]
	opts := make([]T, numItems)

	for i := 0; i < numItems; i++ {
		opts[i] = items[i].item
		doneChans = append(doneChans, items[i].doneChan)
	}

	b.safeDecSizeOfData(b.calcSizeOfData(opts))
	go func() {
		defer func() {
			if r := recover(); r != nil {
				err := fmt.Errorf("panic recovered in flush: %v", r)
				b.l.Error().Msgf("Panic recovered: %v", err)

				// Send error to all done channels
				for _, doneChan := range doneChans {
					select {
					case doneChan <- &FlushResponse[U]{Err: err}:
					default:
						b.l.Error().Msgf("could not send panic error to done chan: %v", err)
					}
				}
			}
		}()

		ctx := context.Background()
		result, err := b.outputFunc(ctx, opts)

		if err != nil {
			for _, doneChan := range doneChans {
				select {
				case doneChan <- &FlushResponse[U]{Err: err}:
				default:
					b.l.Error().Msgf("could not send error to done chan: %v", err)
				}
			}
			return
		}

		for i, d := range doneChans {
			select {
			case d <- &FlushResponse[U]{Result: result[i], Err: nil}:
			default:
				b.l.Error().Msg("could not send done to done chan")
			}
		}

		b.l.Debug().Msgf("flushed %d items", numItems)
	}()
}

func (b *IngestBuf[T, U]) cleanup() error {

	b.lock.Lock()
	b.state = finished
	b.lock.Unlock()

	g := errgroup.Group{}

	for b.safeCheckSizeOfBuffer() > 0 {
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
	defer b.lock.Unlock()

	if b.state == started {
		return nil, fmt.Errorf("buffer already started")
	}
	b.state = started

	go b.buffWorker()

	return b.cleanup, nil
}

func (b *IngestBuf[T, U]) StartDebugLoop() {
	b.l.Debug().Msg("starting debug loop")
	for {
		select {
		case <-time.After(10 * time.Second):
			b.debugBuffer()
		case <-b.ctx.Done():
			b.l.Debug().Msg("stopping debug loop")
			return
		}
	}
}

func (b *IngestBuf[T, U]) BuffItem(item T) (chan *FlushResponse[U], error) {

	if b.state != started {
		return nil, fmt.Errorf("buffer not ready, in state '%v'", b.state.String())
	}

	doneChan := make(chan *FlushResponse[U], 1)

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

func (b *IngestBuf[T, U]) debugBuffer() {

	b.l.Debug().Msgf("============= Buffer =============")
	b.l.Debug().Msgf("%d items", b.safeCheckSizeOfBuffer())
	b.l.Debug().Msgf("%d bytes", b.safeFetchSizeOfData())
	b.l.Debug().Msgf("last flushed at %v", b.safeFetchLastFlush())
	b.l.Debug().Msgf("%d max capacity", b.maxCapacity)
	b.l.Debug().Msgf("%d max data size in queue", b.maxDataSizeInQueue)
	b.l.Debug().Msgf("%v flush period", b.flushPeriod)
	b.l.Debug().Msgf("in state %v", b.state)
	b.l.Debug().Msgf("=====================================")

}
