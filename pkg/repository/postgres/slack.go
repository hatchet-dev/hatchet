package postgres

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

type slackRepository struct {
	*sharedRepository
}

func NewSlackRepository(shared *sharedRepository) repository.SlackRepository {
	return &slackRepository{
		sharedRepository: shared,
	}
}

func (r *slackRepository) UpsertSlackWebhook(ctx context.Context, tenantId string, opts *repository.UpsertSlackWebhookOpts) (*dbsqlc.SlackAppWebhook, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.queries.UpsertSlackWebhook(
		ctx,
		r.pool,
		dbsqlc.UpsertSlackWebhookParams{
			Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
			Teamid:      opts.TeamId,
			Teamname:    opts.TeamName,
			Channelid:   opts.ChannelId,
			Channelname: opts.ChannelName,
			Webhookurl:  opts.WebhookURL,
		},
	)
}

func (r *slackRepository) ListSlackWebhooks(ctx context.Context, tenantId string) ([]*dbsqlc.SlackAppWebhook, error) {
	return r.queries.ListSlackWebhooks(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(tenantId),
	)
}

func (r *slackRepository) GetSlackWebhookById(ctx context.Context, id string) (*dbsqlc.SlackAppWebhook, error) {
	return r.queries.GetSlackWebhookById(
		ctx,
		r.pool,
		sqlchelpers.UUIDFromStr(id),
	)
}

func (r *slackRepository) DeleteSlackWebhook(ctx context.Context, tenantId string, id string) error {
	return r.queries.DeleteSlackWebhook(
		ctx,
		r.pool,
		dbsqlc.DeleteSlackWebhookParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenantId),
			ID:       sqlchelpers.UUIDFromStr(id),
		},
	)
}
