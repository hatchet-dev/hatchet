package transformers

import (
	"fmt"

	"github.com/google/uuid"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

func ToSNSIntegration(sns *db.SNSIntegrationModel, serverUrl string) *gen.SNSIntegration {
	ingestUrl := fmt.Sprintf("%s/api/v1/sns/%s/sns-event", serverUrl, sns.TenantID)

	return &gen.SNSIntegration{
		Metadata:  *toAPIMetadata(sns.ID, sns.CreatedAt, sns.UpdatedAt),
		TopicArn:  sns.TopicArn,
		TenantId:  uuid.MustParse(sns.TenantID),
		IngestUrl: &ingestUrl,
	}
}
