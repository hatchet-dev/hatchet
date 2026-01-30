package transformers

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func ToSNSIntegration(sns *sqlcv1.SNSIntegration, serverUrl string) *gen.SNSIntegration {
	ingestUrl := fmt.Sprintf("%s/api/v1/sns/%s/sns-event", serverUrl, sns.TenantId.String())

	return &gen.SNSIntegration{
		Metadata:  *toAPIMetadata(sns.ID.String(), sns.CreatedAt.Time, sns.UpdatedAt.Time),
		TopicArn:  sns.TopicArn,
		TenantId:  uuid.MustParse(sns.TenantId.String()),
		IngestUrl: &ingestUrl,
	}
}
