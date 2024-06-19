package billing

import (
	"github.com/stripe/stripe-go/v78"

	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type CustomerOpts struct {
	Email string
}

type SubscriptionOpts struct {
	Plan   dbsqlc.TenantSubscriptionPlanCodes
	Period *string
}

type PaymentMethod struct {
	Last4      string
	Brand      stripe.PaymentMethodCardBrand
	Expiration string
}

type Billing interface {
	Enabled() bool
	GetPaymentMethods(tenantId string) ([]*PaymentMethod, error)
	GetCheckoutLink(tenantId string) (*string, error)
	UpsertTenantSubscription(tenant db.TenantModel, opts SubscriptionOpts) (*dbsqlc.TenantSubscription, error)
	GetSubscription(tenantId string) (*dbsqlc.TenantSubscription, error)
	MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error
	VerifyHMACSignature(body []byte, signature string) bool
	HandleUpdateSubscription(id string, planCode string, status string) (*dbsqlc.TenantSubscription, error)
}

type NoOpBilling struct{}

func (b NoOpBilling) Enabled() bool {
	return false
}

func (b NoOpBilling) GetSubscription(tenantId string) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}

func (b NoOpBilling) GetPaymentMethods(tenantId string) ([]*PaymentMethod, error) {
	return nil, nil
}

func (b NoOpBilling) UpsertTenantSubscription(tenant db.TenantModel, opts SubscriptionOpts) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}

func (b NoOpBilling) MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error {
	return nil
}

func (b NoOpBilling) GetCheckoutLink(tenantId string) (*string, error) {
	return nil, nil
}

func (b NoOpBilling) VerifyHMACSignature(body []byte, signature string) bool {
	return false
}

func (b NoOpBilling) HandleUpdateSubscription(id string, planCode string, status string) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}
