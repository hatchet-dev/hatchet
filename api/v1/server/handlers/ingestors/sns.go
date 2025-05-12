package ingestors

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"

	"github.com/hatchet-dev/hatchet/internal/integrations/ingestors/sns"
)

func (i *IngestorsService) SnsUpdate(ctx echo.Context, req gen.SnsUpdateRequestObject) (gen.SnsUpdateResponseObject, error) {
	body, err := io.ReadAll(ctx.Request().Body)

	if err != nil {
		return nil, err
	}

	payload := &sns.Payload{}

	err = json.Unmarshal(body, payload)

	if err != nil {
		return nil, err
	}

	if err := payload.VerifyPayload(); err != nil {
		return nil, err
	}

	tenantId := req.Tenant.String()

	// verify that the tenant and the topic ARN are set in the database
	snsInt, err := i.config.APIRepository.SNS().GetSNSIntegration(ctx.Request().Context(), tenantId, payload.TopicArn)

	if err != nil {
		return nil, err
	}

	if snsInt == nil {
		return nil, fmt.Errorf("SNS integration not found for tenant %s and topic ARN %s", tenantId, payload.TopicArn)
	}

	tenant, err := i.config.APIRepository.Tenant().GetTenantByID(ctx.Request().Context(), tenantId)

	if err != nil {
		return nil, err
	}

	switch payload.Type {
	case "SubscriptionConfirmation":
		_, err := payload.Subscribe()

		if err != nil {
			return nil, err
		}
	case "UnsubscribeConfirmation":
		_, err := payload.Unsubscribe()

		if err != nil {
			return nil, err
		}
	default:
		_, err := i.config.Ingestor.IngestEvent(ctx.Request().Context(), tenant, req.Event, body, nil)

		if err != nil {
			return nil, err
		}
	}

	return gen.SnsUpdate200Response{}, nil
}
