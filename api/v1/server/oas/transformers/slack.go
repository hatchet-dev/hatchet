package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func ToSlackWebhook(slack *dbsqlc.SlackAppWebhook) *gen.SlackWebhook {
	return &gen.SlackWebhook{
		Metadata:    *toAPIMetadata(slack.ID.String(), slack.CreatedAt.Time, slack.UpdatedAt.Time),
		TenantId:    slack.TenantId,
		ChannelId:   slack.ChannelId,
		ChannelName: slack.ChannelName,
		TeamId:      slack.TeamId,
		TeamName:    slack.TeamName,
	}
}
