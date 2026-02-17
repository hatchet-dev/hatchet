package transformers

import (
	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToSlackWebhook(slack *sqlcv1.SlackAppWebhook) *gen.SlackWebhook {
	return &gen.SlackWebhook{
		Metadata:    *toAPIMetadata(slack.ID, slack.CreatedAt.Time, slack.UpdatedAt.Time),
		TenantId:    slack.TenantId,
		ChannelId:   slack.ChannelId,
		ChannelName: slack.ChannelName,
		TeamId:      slack.TeamId,
		TeamName:    slack.TeamName,
	}
}
