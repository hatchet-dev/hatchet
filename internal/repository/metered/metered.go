package metered

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/dbsqlc"
)

type Metered struct {
	entitlements repository.EntitlementsRepository
	l            *zerolog.Logger
}

func NewMetered(entitlements repository.EntitlementsRepository, l *zerolog.Logger) *Metered {
	return &Metered{
		entitlements: entitlements,
		l:            l,
	}
}

var ErrResourceExhausted = fmt.Errorf("resource exhausted")

func MakeMetered[T any](ctx context.Context, m *Metered, resource dbsqlc.LimitResource, tenantId string, f func() (*T, error)) (*T, error) {

	canCreate, err := m.entitlements.TenantLimit().CanCreate(ctx, resource, tenantId)

	if err != nil {
		return nil, fmt.Errorf("could not check tenant limit: %w", err)
	}

	if !canCreate {
		return nil, ErrResourceExhausted
	}

	res, err := f()

	if err != nil {
		return nil, err
	}

	deferredMeter := func() {
		_, err := m.entitlements.TenantLimit().Meter(ctx, resource, tenantId)

		// TODO: we should probably publish an event here if limits are exhausted to notify immediately

		if err != nil {
			m.l.Error().Err(err).Msg("could not meter resource")
		}
	}

	defer deferredMeter()

	return res, nil
}
