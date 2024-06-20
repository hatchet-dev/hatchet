package prisma

import (
	"context"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

type slackRepository struct {
	client *db.PrismaClient
	v      validator.Validator
}

func NewSlackRepository(client *db.PrismaClient, v validator.Validator) repository.SlackRepository {
	return &slackRepository{
		client: client,
		v:      v,
	}
}

func (r *slackRepository) UpsertSlackWebhook(tenantId string, opts *repository.UpsertSlackWebhookOpts) (*db.SlackAppWebhookModel, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	return r.client.SlackAppWebhook.UpsertOne(
		db.SlackAppWebhook.TenantIDTeamIDChannelID(
			db.SlackAppWebhook.TenantID.Equals(tenantId),
			db.SlackAppWebhook.TeamID.Equals(opts.TeamId),
			db.SlackAppWebhook.ChannelID.Equals(opts.ChannelId),
		),
	).Create(
		db.SlackAppWebhook.Tenant.Link(
			db.Tenant.ID.Equals(tenantId),
		),
		db.SlackAppWebhook.TeamID.Set(opts.TeamId),
		db.SlackAppWebhook.TeamName.Set(opts.TeamName),
		db.SlackAppWebhook.ChannelID.Set(opts.ChannelId),
		db.SlackAppWebhook.ChannelName.Set(opts.ChannelName),
		db.SlackAppWebhook.WebhookURL.Set(opts.WebhookURL),
	).Update(
		db.SlackAppWebhook.TeamName.Set(opts.TeamName),
		db.SlackAppWebhook.ChannelName.Set(opts.ChannelName),
		db.SlackAppWebhook.WebhookURL.Set(opts.WebhookURL),
	).Exec(context.Background())
}

func (r *slackRepository) ListSlackWebhooks(tenantId string) ([]db.SlackAppWebhookModel, error) {
	return r.client.SlackAppWebhook.FindMany(
		db.SlackAppWebhook.TenantID.Equals(tenantId),
	).Exec(context.Background())
}

func (r *slackRepository) GetSlackWebhookById(id string) (*db.SlackAppWebhookModel, error) {
	return r.client.SlackAppWebhook.FindUnique(
		db.SlackAppWebhook.ID.Equals(id),
	).Exec(context.Background())
}

func (r *slackRepository) DeleteSlackWebhook(tenantId string, id string) error {
	_, err := r.client.SlackAppWebhook.FindUnique(
		db.SlackAppWebhook.ID.Equals(id),
	).Delete().Exec(context.Background())

	return err
}
