package ingestor

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
)

type IngestBuf struct {
	maxCapacity    int
	flushPeriod    time.Duration
	eventOpsChan   chan *eventBuffWrapper
	lastFlush      time.Time
	internalArr    []*eventBuffWrapper
	sizeOfData     int
	BulkCreateFunc func(ctx context.Context, opts *repository.BulkCreateEventSharedTenantOpts) (*repository.BulkCreateEventResult, error)
}

type eventBuffWrapper struct {
	eventOps *repository.CreateEventOpts
	doneChan chan *flushResponse
}

const MAX_SIZE = 1000000

func (b *IngestBuf) buffEventWorker(ctx context.Context) {
	for {
		select {
		case e := <-b.eventOpsChan:
			b.internalArr = append(b.internalArr, e)

			if len(b.internalArr) >= b.maxCapacity {
				fmt.Println("Buffer is full number of messages is > ", b.maxCapacity)
				b.flush()
				fmt.Println("Finished flushing buffer")
			}
			if b.sizeOfData >= MAX_SIZE {
				fmt.Println("Buffer is full size of data is >", b.sizeOfData)

				b.flush()
				fmt.Println("Finished flushing buffer cause of size")
			}

			// buff is full flush to BulkCreateEvent
		case <-time.After(time.Until(b.lastFlush.Add(b.flushPeriod))):
			if len(b.internalArr) > 0 {
				fmt.Println("===========================Time is up and we have at least one event")
				b.flush()
			} else {
				b.lastFlush = time.Now()
				fmt.Println("Time is up and we have no events")

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
func (i *IngestorImpl) StartBuffer(ctx context.Context) {
	fmt.Println("============================================  Starting buffer")
	i.buff.eventOpsChan = make(chan *eventBuffWrapper, 2*i.buff.maxCapacity)
	i.buff.BulkCreateFunc = i.eventRepository.BulkCreateEventSharedTenant
	i.buff.Start(ctx)
}
