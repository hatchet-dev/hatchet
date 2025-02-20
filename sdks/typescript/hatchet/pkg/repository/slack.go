package repository

import "github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"

type UpsertSlackWebhookOpts struct {
	TeamId string `validate:"required,min=1,max=255"`

	TeamName string `validate:"required,min=1,max=255"`

	ChannelId string `validate:"required,min=1,max=255"`

	ChannelName string `validate:"required,min=1,max=255"`

	WebhookURL []byte `validate:"required,min=1"`
}

type SlackRepository interface {
	UpsertSlackWebhook(tenantId string, opts *UpsertSlackWebhookOpts) (*db.SlackAppWebhookModel, error)

	ListSlackWebhooks(tenantId string) ([]db.SlackAppWebhookModel, error)

	GetSlackWebhookById(id string) (*db.SlackAppWebhookModel, error)

	DeleteSlackWebhook(tenantId string, id string) error
}
