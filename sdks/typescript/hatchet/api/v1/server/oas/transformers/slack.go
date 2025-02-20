package transformers

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToSlackWebhook(slack *db.SlackAppWebhookModel) *gen.SlackWebhook {
	return &gen.SlackWebhook{
		Metadata:    *toAPIMetadata(slack.ID, slack.CreatedAt, slack.UpdatedAt),
		TenantId:    uuid.MustParse(slack.TenantID),
		ChannelId:   slack.ChannelID,
		ChannelName: slack.ChannelName,
		TeamId:      slack.TeamID,
		TeamName:    slack.TeamName,
	}
}
