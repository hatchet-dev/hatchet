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
	PlanCode string
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
	UpsertTenantSubscription(tenant db.TenantModel, opts *SubscriptionOpts) (*dbsqlc.TenantSubscription, error)
	GetSubscription(tenantId string) (*dbsqlc.TenantSubscription, error)
	MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error
}

type NoOpBilling struct{}

func (a NoOpBilling) Enabled() bool {
	return false
}

func (a NoOpBilling) GetSubscription(tenantId string) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}

func (a NoOpBilling) GetPaymentMethods(tenantId string) ([]*PaymentMethod, error) {
	return nil, nil
}

func (a NoOpBilling) UpsertTenantSubscription(tenant db.TenantModel, opts *SubscriptionOpts) (*dbsqlc.TenantSubscription, error) {
	return nil, nil
}

func (a NoOpBilling) MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error {
	return nil
}

func (a NoOpBilling) GetCheckoutLink(tenantId string) (*string, error) {
	return nil, nil
}
