package lago

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/getlago/lago-go-client"
	"github.com/rs/zerolog"
	"github.com/stripe/stripe-go/v78"
	"github.com/stripe/stripe-go/v78/client"

	"github.com/hatchet-dev/hatchet/pkg/billing"
	"github.com/hatchet-dev/hatchet/pkg/config/shared"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

type LagoBilling struct {
	client    *lago.Client
	l         *zerolog.Logger
	e         repository.EntitlementsRepository
	stripe    *client.API
	serverURL string
	ApiKey    string
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
		ApiKey:    opts.ApiKey,
	}, nil
}

func (l *LagoBilling) Enabled() bool {
	return true
}

func (l *LagoBilling) GetCustomer(tenantId string) (*lago.Customer, *lago.Error) {
	c, err := l.client.Customer().Get(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	return c, nil
}

func (l *LagoBilling) GetPaymentMethods(tenantId string) ([]*billing.PaymentMethod, error) {
	c, err := l.GetCustomer(tenantId)

	if err != nil {

		if err.HTTPStatusCode == 404 {
			return nil, nil
		}

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

	customer, lagoErr := l.GetCustomer(tenantId)

	if lagoErr != nil {

		if lagoErr.HTTPStatusCode == 404 {
			return nil, nil
		}

		return nil, lagoErr
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

func (l *LagoBilling) UpsertTenantSubscription(tenant db.TenantModel, opts billing.SubscriptionOpts) (*dbsqlc.TenantSubscription, error) {
	ctx := context.Background()

	customer, lagoErr := l.GetCustomer(tenant.ID)

	if lagoErr != nil {
		if lagoErr.HTTPStatusCode == 404 {
			customer = nil
		} else {
			return nil, lagoErr
		}
	}

	if customer == nil {
		_, err := l.UpsertTenant(tenant)

		if err != nil {
			return nil, err
		}
	}

	planCode := string(opts.Plan)

	if opts.Plan != dbsqlc.TenantSubscriptionPlanCodesFree && opts.Period == nil {
		return nil, fmt.Errorf("period is required for non-free plans")
	}

	if opts.Period != nil {
		planCode = fmt.Sprintf("%s:%s", string(opts.Plan), *opts.Period)
	}

	sub, subErr := l.client.Subscription().Create(ctx, &lago.SubscriptionInput{
		ExternalCustomerID: tenant.ID,
		ExternalID:         tenant.ID,
		PlanCode:           planCode,
		BillingTime:        lago.Anniversary,
	})

	if subErr != nil {
		return nil, subErr
	}

	var note string

	if sub.NextPlanCode != "" {

		downgradeDate, err := time.Parse("2006-01-02", sub.DowngradePlanDate)
		if err != nil {
			return nil, err
		}
		formattedDate := downgradeDate.Format("January 2, 2006")

		note = fmt.Sprintf("Downgrading to %s on %s", sub.NextPlanCode, formattedDate)
	}

	s, err := l.HandleUpdateSubscription(tenant.ID, sub.PlanCode, string(sub.Status), note)

	if err != nil {
		return nil, err
	}

	return s, nil
}

func (l *LagoBilling) HandleUpdateSubscription(id string, planCode string, status string, note string) (*dbsqlc.TenantSubscription, error) {
	ctx := context.Background()

	planCodeParts := strings.Split(planCode, ":")
	plan := dbsqlc.TenantSubscriptionPlanCodes(planCodeParts[0])

	period := dbsqlc.NullTenantSubscriptionPeriod{}

	if len(planCodeParts) > 1 {
		period = dbsqlc.NullTenantSubscriptionPeriod{
			TenantSubscriptionPeriod: dbsqlc.TenantSubscriptionPeriod(planCodeParts[1]),
			Valid:                    true,
		}
	}

	_, s, err := l.e.TenantSubscription().UpsertSubscription(ctx,
		dbsqlc.UpsertTenantSubscriptionParams{
			Tenantid: sqlchelpers.UUIDFromStr(id),
			Plan: dbsqlc.NullTenantSubscriptionPlanCodes{
				TenantSubscriptionPlanCodes: plan,
				Valid:                       true,
			},
			Period: period,
			Status: dbsqlc.NullTenantSubscriptionStatus{
				TenantSubscriptionStatus: dbsqlc.TenantSubscriptionStatus(status),
				Valid:                    true,
			},
			Note: sqlchelpers.TextFromStr(note),
		})

	if err != nil {
		return nil, fmt.Errorf("failed to upsert subscription: %w", err)
	}

	// Update Tenant Limits
	err = l.e.TenantLimit().UpsertTenantLimits(ctx, id, &plan)

	if err != nil {
		return nil, fmt.Errorf("failed to upsert default limits: %w", err)
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

func (l *LagoBilling) VerifyHMACSignature(body []byte, signature string) bool {
	h := hmac.New(sha256.New, []byte(l.ApiKey))
	h.Write(body)
	calcSig := h.Sum(nil)
	base64Sig := base64.StdEncoding.EncodeToString(calcSig)
	return signature == base64Sig
}

func (l *LagoBilling) Plans() ([]*billing.Plan, error) {
	plans, err := l.client.Plan().GetList(context.Background(), &lago.PlanListInput{
		PerPage: 10,
		Page:    1,
	})

	if err != nil {
		return nil, err
	}

	var result []*billing.Plan

	for _, plan := range plans.Plans {
		p := string(plan.Interval)
		result = append(result, &billing.Plan{
			PlanCode:    plan.Code,
			Name:        plan.Name,
			Description: plan.Description,
			AmountCents: plan.AmountCents,
			Period:      &p,
		})
	}

	return result, nil
}
