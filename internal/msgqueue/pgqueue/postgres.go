package pgqueue

import (
	"context"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
)

type Queue struct {
}

func (q Queue) Name() string {
	// TODO implement me
	panic("implement me")
}

func (q Queue) Durable() bool {
	// TODO implement me
	panic("implement me")
}

func (q Queue) AutoDeleted() bool {
	// TODO implement me
	panic("implement me")
}

func (q Queue) Exclusive() bool {
	// TODO implement me
	panic("implement me")
}

func (q Queue) FanoutExchangeKey() string {
	// TODO implement me
	panic("implement me")
}

func (q Queue) DLX() string {
	// TODO implement me
	panic("implement me")
}

type Message struct {
}

func (m Message) AddMessage(ctx context.Context, queue msgqueue.Queue, task *msgqueue.Message) error {
	// TODO implement me
	panic("implement me")
}

func (m Message) Subscribe(queue msgqueue.Queue, preAck msgqueue.AckHook, postAck msgqueue.AckHook) (func() error, error) {
	// TODO implement me
	panic("implement me")
}

func (m Message) RegisterTenant(ctx context.Context, tenantId string) error {
	// TODO implement me
	panic("implement me")
}

func (m Message) IsReady() bool {
	// TODO implement me
	panic("implement me")
}
