package transformers

import (
	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToSlackWebhook(slack *dbsqlc.SlackAppWebhook) *gen.SlackWebhook {
	return &gen.SlackWebhook{
		Metadata:    *toAPIMetadata(sqlchelpers.UUIDToStr(slack.ID), slack.CreatedAt.Time, slack.UpdatedAt.Time),
		TenantId:    uuid.MustParse(sqlchelpers.UUIDToStr(slack.TenantId)),
		ChannelId:   slack.ChannelId,
		ChannelName: slack.ChannelName,
		TeamId:      slack.TeamId,
		TeamName:    slack.TeamName,
	}
}
