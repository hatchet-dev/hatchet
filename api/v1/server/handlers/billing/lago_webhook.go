package billing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/getlago/lago-go-client"
	"github.com/labstack/echo/v4"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
)

// Define the base struct for the common fields
type BaseMessage struct {
	WebhookType  string                 `json:"webhook_type"`
	ObjectType   string                 `json:"object_type"`
	Subscription map[string]interface{} `json:"subscription,omitempty"`
}

func (b *BillingService) LagoMessageCreate(ctx echo.Context, req gen.LagoMessageCreateRequestObject) (gen.LagoMessageCreateResponseObject, error) {
	body, err := io.ReadAll(ctx.Request().Body)
	if err != nil {
		return gen.LagoMessageCreate403JSONResponse{}, err
	}

	// Verify the signature
	b.config.Billing.VerifyHMACSignature(body, ctx.Request().Header.Get("X-Lago-Signature"))

	// Unmarshal the JSON body into the base struct
	var baseMsg BaseMessage
	if err := json.Unmarshal(body, &baseMsg); err != nil {
		return gen.LagoMessageCreate400JSONResponse{}, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Handle different object types based on the object_type field
	switch baseMsg.ObjectType {
	case "subscription":
		var subscription lago.Subscription
		if objBytes, err := json.Marshal(baseMsg.Subscription); err == nil {
			if err := json.Unmarshal(objBytes, &subscription); err != nil {
				return gen.LagoMessageCreate400JSONResponse{}, echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
			}
		}

		switch baseMsg.WebhookType {
		case "subscription.started":
			err = b.handleSubscriptionChange(subscription)
		case "subscription.terminated":
			err = b.handleSubscriptionChange(subscription)
		}
	case "invoice":
		// TODO "webhook_type": "invoice.payment_failure",
		break

	}

	if err != nil {
		return nil, err
	}

	return gen.LagoMessageCreate200Response{}, nil
}

func (b *BillingService) handleSubscriptionChange(subscription lago.Subscription) error {
	b.config.Logger.Info().Msgf("handling subscription id %s", subscription.ExternalID)

	planCode := subscription.PlanCode
	id := subscription.ExternalCustomerID

	status := subscription.Status

	// TODO handle next plan code
	// nextPlanCode := subscription.NextPlanCode
	// downgradePlanDate := subscription.DowngradePlanDate
	// Also on termination, we need to handle downgrading limits

	_, err := b.config.Billing.HandleUpdateSubscription(id, planCode, string(status))

	if err != nil {
		return fmt.Errorf("error updating subscription: %v", err)
	}

	return nil
}
