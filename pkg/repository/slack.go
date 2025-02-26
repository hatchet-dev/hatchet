package repository

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

type UpsertSlackWebhookOpts struct {
	TeamId string `validate:"required,min=1,max=255"`

	TeamName string `validate:"required,min=1,max=255"`

	ChannelId string `validate:"required,min=1,max=255"`

	ChannelName string `validate:"required,min=1,max=255"`

	WebhookURL []byte `validate:"required,min=1"`
}

type SlackRepository interface {
	UpsertSlackWebhook(ctx context.Context, tenantId string, opts *UpsertSlackWebhookOpts) (*dbsqlc.SlackAppWebhook, error)

	ListSlackWebhooks(ctx context.Context, tenantId string) ([]*dbsqlc.SlackAppWebhook, error)

	GetSlackWebhookById(ctx context.Context, id string) (*dbsqlc.SlackAppWebhook, error)

	DeleteSlackWebhook(ctx context.Context, tenantId string, id string) error
}
