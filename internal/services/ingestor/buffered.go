package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type IngestBuf struct {
	maxCapacity        int           // max number of events to hold in buffer before we flush
	flushPeriod        time.Duration // max time to hold events in buffer before we flush
	eventOpsChan       chan *eventBuffWrapper
	lastFlush          time.Time
	internalArr        []*eventBuffWrapper
	sizeOfData         int // size of data in buffer
	maxDataSizeInQueue int // max number of bytes to hold in buffer before we flush
	BulkCreateFunc     func(ctx context.Context, opts *repository.BulkCreateEventSharedTenantOpts) (*repository.BulkCreateEventResult, error)
}

func NewIngestBuffer(maxCapacity int, flushPeriod time.Duration, maxDataSizeInQueue int, bulkCreateFunc func(ctx context.Context, opts *repository.BulkCreateEventSharedTenantOpts) (*repository.BulkCreateEventResult, error)) *IngestBuf {
	return &IngestBuf{
		maxCapacity:        maxCapacity,
		flushPeriod:        flushPeriod,
		eventOpsChan:       make(chan *eventBuffWrapper, maxCapacity*2),
		lastFlush:          time.Now(),
		internalArr:        make([]*eventBuffWrapper, 0),
		sizeOfData:         0,
		maxDataSizeInQueue: maxDataSizeInQueue,
		BulkCreateFunc:     bulkCreateFunc,
	}
}

type eventBuffWrapper struct {
	eventOps *repository.CreateEventOpts
	doneChan chan *flushResponse
}

func (b *IngestBuf) buffEventWorker(ctx context.Context) {
	for {
		select {
		case e := <-b.eventOpsChan:
			b.internalArr = append(b.internalArr, e)
			b.sizeOfData += len(e.eventOps.Data)
			b.sizeOfData += len(e.eventOps.AdditionalMetadata)

			if len(b.internalArr) >= b.maxCapacity {
				b.flush()
			}
			if b.sizeOfData >= b.maxDataSizeInQueue {
				b.flush()
			}

		case <-time.After(time.Until(b.lastFlush.Add(b.flushPeriod))):
			if len(b.internalArr) > 0 {
				b.flush()
			} else {
				b.lastFlush = time.Now()

			}
		case <-ctx.Done():
			return
		}

	}

}

type flushResponse struct {
	event *dbsqlc.Event
	err   error
}

func (b *IngestBuf) flush() {
	// flush to BulkCreateEvent

	var events []*eventBuffWrapper

	if len(b.internalArr) >= b.maxCapacity {
		events = b.internalArr[:b.maxCapacity]

		b.internalArr = b.internalArr[b.maxCapacity:]
	} else {
		events = b.internalArr

		b.internalArr = nil
	}

	numEvents := len(events)
	eventOpts := make([]*repository.CreateEventOpts, numEvents)

	var doneChans []chan *flushResponse

	for i := 0; i < numEvents; i++ {
		eventOpts[i] = events[i].eventOps
		doneChans = append(doneChans, events[i].doneChan)
	}

	ctx := context.Background()

	// bulk create events
	writtenEvents, err := b.BulkCreateFunc(ctx, &repository.BulkCreateEventSharedTenantOpts{
		Events: eventOpts,
	})

	fmt.Printf("============================================  BulkCreateEventSharedTenant flushed %+v events\n", len(writtenEvents.Events))

	if err != nil {
		for _, doneChan := range doneChans {
			doneChan <- &flushResponse{event: nil, err: nil}
		}
		return
	}

	for i, d := range doneChans {
		// FIXME: these events are not in the correct order - need to fix cause we could send the wrong event back to the wrong tenant
		d <- &flushResponse{event: writtenEvents.Events[i], err: nil}
	}
	b.lastFlush = time.Now()
	// TODO: if we have additional events in the buffer, that haven't been flushed, we will not be counting their size
	b.sizeOfData = 0

	fmt.Println("============================================  Done sending to done chans")

}

func (b *IngestBuf) Start(ctx context.Context) {
	go b.buffEventWorker(ctx)
}
