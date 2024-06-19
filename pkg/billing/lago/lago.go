package lago

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/getlago/lago-go-client"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/config/shared"
	"github.com/hatchet-dev/hatchet/internal/logger"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/billing"
)

type LagoBilling struct {
	client *lago.Client
	l      *zerolog.Logger
	e      repository.EntitlementsRepository
}

type LagoBillingOpts struct {
	ApiKey  string
	BaseUrl string
	Logger  shared.LoggerConfigFile
}

func NewLagoBilling(opts *LagoBillingOpts, e *repository.EntitlementsRepository) (*LagoBilling, error) {
	if opts.ApiKey == "" || opts.BaseUrl == "" {
		return nil, fmt.Errorf("api key and base url are required if lago is enabled")
	}

	lagoClient := lago.New().SetBaseURL(opts.BaseUrl).SetApiKey(opts.ApiKey)

	l := logger.NewStdErr(&opts.Logger, "billing")

	return &LagoBilling{
		client: lagoClient,
		l:      &l,
		e:      *e,
	}, nil
}

func (l *LagoBilling) Enabled() bool {
	return true
}

func (l *LagoBilling) GetTenant(tenantId string) (*lago.Customer, error) {
	customer, err := l.client.Customer().Get(context.Background(), tenantId)

	if err != nil {
		return nil, err
	}

	return customer, nil
}

func (l *LagoBilling) UpsertTenant(tenant db.TenantModel) (*lago.Customer, error) {
	customer, err := l.client.Customer().Update(context.Background(), &lago.CustomerInput{
		ExternalID: tenant.ID,
	})

	if err != nil {
		return nil, err
	}

	return customer, nil
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
