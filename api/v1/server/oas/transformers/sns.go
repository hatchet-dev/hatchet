package transformers

import (
	"fmt"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/dbsqlc"
)

func ToSNSIntegration(sns *dbsqlc.SNSIntegration, serverUrl string) *gen.SNSIntegration {
	ingestUrl := fmt.Sprintf("%s/api/v1/sns/%s/sns-event", serverUrl, sns.TenantId.String())

	return &gen.SNSIntegration{
		Metadata:  *toAPIMetadata(sns.ID.String(), sns.CreatedAt.Time, sns.UpdatedAt.Time),
		TopicArn:  sns.TopicArn,
		TenantId:  sns.TenantId,
		IngestUrl: &ingestUrl,
	}
}
