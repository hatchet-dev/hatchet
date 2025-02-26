package transformers

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
)

func ToSNSIntegration(sns *dbsqlc.SNSIntegration, serverUrl string) *gen.SNSIntegration {
	ingestUrl := fmt.Sprintf("%s/api/v1/sns/%s/sns-event", serverUrl, sqlchelpers.UUIDToStr(sns.TenantId))

	return &gen.SNSIntegration{
		Metadata:  *toAPIMetadata(sqlchelpers.UUIDToStr(sns.ID), sns.CreatedAt.Time, sns.UpdatedAt.Time),
		TopicArn:  sns.TopicArn,
		TenantId:  uuid.MustParse(sqlchelpers.UUIDToStr(sns.TenantId)),
		IngestUrl: &ingestUrl,
	}
}
