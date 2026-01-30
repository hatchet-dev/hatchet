package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type UpsertSlackWebhookOpts struct {
	TeamId string `validate:"required,min=1,max=255"`

	TeamName string `validate:"required,min=1,max=255"`

	ChannelId string `validate:"required,min=1,max=255"`

	ChannelName string `validate:"required,min=1,max=255"`

	WebhookURL []byte `validate:"required,min=1"`
}

type SlackRepository interface {
	UpsertSlackWebhook(ctx context.Context, tenantId string, opts *UpsertSlackWebhookOpts) (*sqlcv1.SlackAppWebhook, error)

	ListSlackWebhooks(ctx context.Context, tenantId string) ([]*sqlcv1.SlackAppWebhook, error)

	GetSlackWebhookById(ctx context.Context, id string) (*sqlcv1.SlackAppWebhook, error)

	DeleteSlackWebhook(ctx context.Context, tenantId string, id string) error
}

type slackRepository struct {
	*sharedRepository
}

func newSlackRepository(shared *sharedRepository) SlackRepository {
	return &slackRepository{
		sharedRepository: shared,
	}
}

func (r *slackRepository) UpsertSlackWebhook(ctx context.Context, tenantId string, opts *UpsertSlackWebhookOpts) (*sqlcv1.SlackAppWebhook, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.UpsertSlackWebhook(
		ctx,
		r.pool,
		sqlcv1.UpsertSlackWebhookParams{
			Tenantid:    uuid.MustParse(tenantId),
			Teamid:      opts.TeamId,
			Teamname:    opts.TeamName,
			Channelid:   opts.ChannelId,
			Channelname: opts.ChannelName,
			Webhookurl:  opts.WebhookURL,
		},
	)
}

func (r *slackRepository) ListSlackWebhooks(ctx context.Context, tenantId string) ([]*sqlcv1.SlackAppWebhook, error) {
	return r.queries.ListSlackWebhooks(
		ctx,
		r.pool,
		uuid.MustParse(tenantId),
	)
}

func (r *slackRepository) GetSlackWebhookById(ctx context.Context, id string) (*sqlcv1.SlackAppWebhook, error) {
	return r.queries.GetSlackWebhookById(
		ctx,
		r.pool,
		uuid.MustParse(id),
	)
}

func (r *slackRepository) DeleteSlackWebhook(ctx context.Context, tenantId string, id string) error {
	return r.queries.DeleteSlackWebhook(
		ctx,
		r.pool,
		sqlcv1.DeleteSlackWebhookParams{
			Tenantid: uuid.MustParse(tenantId),
			ID:       uuid.MustParse(id),
		},
	)
}
