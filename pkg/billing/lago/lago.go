package lago

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/getlago/lago-go-client"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"

	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/billing"
)

type LagoBilling struct {
	client    *lago.Client
	l         *zerolog.Logger
	e         repository.EntitlementsRepository
	stripe    *client.API
	serverURL string
}

type LagoBillingOpts struct {
	ApiKey    string
	BaseUrl   string
	StripeKey string
	Logger    shared.LoggerConfigFile
}

func NewLagoBilling(opts *LagoBillingOpts, e *repository.EntitlementsRepository, serverUrl string) (*LagoBilling, error) {
	if opts.ApiKey == "" || opts.BaseUrl == "" {
		return nil, fmt.Errorf("api key and base url are required if lago is enabled")
	}

	lagoClient := lago.New().SetBaseURL(opts.BaseUrl).SetApiKey(opts.ApiKey)

	l := logger.NewStdErr(&opts.Logger, "billing")

	stripe := &client.API{}

	stripe.Init(opts.StripeKey, nil)

	return &LagoBilling{
		client:    lagoClient,
		l:         &l,
		e:         *e,
		stripe:    stripe,
		serverURL: serverUrl,
	}, nil
}

func (l *LagoBilling) Enabled() bool {
	return true
}

func (l *LagoBilling) GetCustomer(tenantId string) (*lago.Customer, error) {
	c, err := l.client.Customer().Get(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (l *LagoBilling) GetPaymentMethods(tenantId string) ([]*billing.PaymentMethod, error) {
	c, err := l.GetCustomer(tenantId)

	if err != nil {
		return nil, err
	}

	stripeId := c.BillingConfiguration.ProviderCustomerID

	params := &stripe.PaymentMethodListParams{
		Customer: stripe.String(stripeId),
	}

	params.Limit = stripe.Int64(3)
	result := l.stripe.PaymentMethods.List(params)

	var methods []*billing.PaymentMethod

	for result.Next() {
		method := result.PaymentMethod()

		exp := fmt.Sprintf("%d/%d", method.Card.ExpMonth, method.Card.ExpYear)

		paymentMethod := &billing.PaymentMethod{
			Last4:      method.Card.Last4,
			Brand:      method.Card.Brand,
			Expiration: exp,
		}
		methods = append(methods, paymentMethod)
	}

	return methods, nil
}

func (l *LagoBilling) UpsertTenant(tenant db.TenantModel) (*lago.Customer, error) {
	customer, err := l.client.Customer().Update(context.Background(), &lago.CustomerInput{
		ExternalID: tenant.ID,
		Name:       tenant.Name,
		BillingConfiguration: lago.CustomerBillingConfigurationInput{
			PaymentProvider:  lago.PaymentProviderStripe,
			Sync:             true,
			SyncWithProvider: true,
		},
	})

	if err != nil {
		return nil, err
	}

	return customer, nil
}

func (l *LagoBilling) GetCheckoutLink(tenantId string) (*string, error) {
	// link, err := l.client.Customer().CheckoutUrl(context.Background(), tenantId)

	customer, err := l.GetCustomer(tenantId)

	if err != nil {
		return nil, err
	}

	returnUrl := fmt.Sprintf("%s/tenant-settings/billing-and-limits?tenant=%s", l.serverURL, tenantId)

	res, err := l.stripe.BillingPortalSessions.New(&stripe.BillingPortalSessionParams{
		Customer:  &customer.BillingConfiguration.ProviderCustomerID,
		ReturnURL: &returnUrl,
	})

	if err != nil {
		return nil, err
	}

	return &res.URL, nil
}

func (l *LagoBilling) GetSubscription(tenantId string) (*dbsqlc.TenantSubscription, error) {
	ctx := context.Background()

	sub, err := l.e.TenantSubscription().GetSubscription(ctx, tenantId)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}

	return sub, nil
}

func (l *LagoBilling) UpsertTenantSubscription(tenant db.TenantModel, opts *billing.SubscriptionOpts) (*dbsqlc.TenantSubscription, error) {
	ctx := context.Background()

	_, lagoErr := l.UpsertTenant(tenant)

	if lagoErr != nil {
		return nil, lagoErr
	}

	sub, subErr := l.client.Subscription().Create(ctx, &lago.SubscriptionInput{
		ExternalCustomerID: tenant.ID,
		ExternalID:         tenant.ID,
		PlanCode:           opts.PlanCode,
		BillingTime:        lago.Anniversary,
	})

	if subErr != nil {
		return nil, subErr
	}

	_, s, err := l.e.TenantSubscription().UpsertSubscription(ctx,
		dbsqlc.UpsertTenantSubscriptionParams{
			Tenantid: sqlchelpers.UUIDFromStr(tenant.ID),
			PlanCode: sqlchelpers.TextFromStr(sub.PlanCode),
			Status: dbsqlc.NullTenantSubscriptionStatus{
				TenantSubscriptionStatus: dbsqlc.TenantSubscriptionStatus(sub.Status),
				Valid:                    true,
			},
		})

	if err != nil {
		return nil, fmt.Errorf("failed to upsert subscription: %w", err)
	}

	return s, nil
}

func (l *LagoBilling) MeterMetric(tenantId string, resource dbsqlc.LimitResource, uniqueId string, limitVal *int32) error {
	event := lago.EventInput{
		TransactionID:          uniqueId,
		ExternalSubscriptionID: tenantId,
		Code:                   string(resource),
		Timestamp:              strconv.FormatInt(time.Now().Unix(), 10),
		Properties: map[string]interface{}{
			"limit_val": limitVal,
		},
	}

	_, err := l.client.Event().Create(context.Background(), &event)

	if err != nil {
		return err
	}

	return nil
}
