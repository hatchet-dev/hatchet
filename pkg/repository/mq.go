package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type PubMessage struct {
	Channel string
	Payload string
}

type MessageQueueRepository interface {
	// PubSub
	Listen(ctx context.Context, name string, f func(ctx context.Context, notification *PubMessage) error) error
	Notify(ctx context.Context, name string, payload string) error

	// Queues
	BindQueue(ctx context.Context, queue string, durable, autoDeleted, exclusive bool, exclusiveConsumer *string) error
	UpdateQueueLastActive(ctx context.Context, queue string) error
	CleanupQueues(ctx context.Context) error

	// Messages
	AddMessage(ctx context.Context, queue string, payload []byte) error
	ReadMessages(ctx context.Context, queue string, qos int) ([]*dbsqlc.ReadMessagesRow, error)
	AckMessage(ctx context.Context, id int64) error
	CleanupMessageQueueItems(ctx context.Context) error
}
