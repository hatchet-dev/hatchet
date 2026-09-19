package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

const (
	defaultServerlessRequestTimeoutSeconds int32 = 60
	defaultServerlessPollIntervalSeconds   int32 = 30
	defaultServerlessInlineWaitBudgetMs    int32 = 5000
)

type CreateServerlessEndpointOpts struct {
	Name string `validate:"required,hatchetName"`

	// Kind defaults to CLOUDFLARE_WORKERS.
	Kind *sqlcv1.V1ServerlessEndpointKind `validate:"omitnil,oneof=GENERIC_HTTP CLOUDFLARE_WORKERS"`

	HealthcheckUrl string `validate:"required,url"`
	TriggerUrl     string `validate:"required,url"`

	// SigningSecretEnc is the signing secret already encrypted with
	// contract.SigningSecretEncryptionDataID. The repository never sees the plaintext.
	SigningSecretEnc string `validate:"required"`

	RequestTimeoutSeconds *int32 `validate:"omitnil,min=1,max=600"`
	PollIntervalSeconds   *int32 `validate:"omitnil,min=5,max=3600"`
	InlineWaitBudgetMs    *int32 `validate:"omitnil,min=0,max=600000"`

	// Labels is a JSON object; nil means {}.
	Labels []byte

	// Enabled defaults to true.
	Enabled *bool
}

type UpdateServerlessEndpointOpts struct {
	Name                  *string                          `validate:"omitnil,hatchetName"`
	Kind                  *sqlcv1.V1ServerlessEndpointKind `validate:"omitnil,oneof=GENERIC_HTTP CLOUDFLARE_WORKERS"`
	HealthcheckUrl        *string                          `validate:"omitnil,url"`
	TriggerUrl            *string                          `validate:"omitnil,url"`
	SigningSecretEnc      *string                          `validate:"omitnil,min=1"`
	RequestTimeoutSeconds *int32                           `validate:"omitnil,min=1,max=600"`
	PollIntervalSeconds   *int32                           `validate:"omitnil,min=5,max=3600"`
	InlineWaitBudgetMs    *int32                           `validate:"omitnil,min=0,max=600000"`
	Labels                []byte
	Enabled               *bool
}

type ListServerlessEndpointsOpts struct {
	Limit  int64 `validate:"min=1,max=1000"`
	Offset int64 `validate:"min=0"`
}

type serverlessEndpointRepository struct {
	*sharedRepository
}

func int32OrDefault(v *int32, def int32) int32 {
	if v == nil {
		return def
	}

	return *v
}

// validateLabels checks that labels, when given, is a JSON object. The validator's json tag
// only handles strings, so this is done by hand.
func validateLabels(labels []byte) ([]byte, error) {
	if len(labels) == 0 {
		return []byte(`{}`), nil
	}

	var obj map[string]any

	if err := json.Unmarshal(labels, &obj); err != nil {
		return nil, fmt.Errorf("labels must be a JSON object: %w", err)
	}

	return labels, nil
}

func (r *serverlessEndpointRepository) Create(ctx context.Context, tenantId uuid.UUID, opts CreateServerlessEndpointOpts) (*sqlcv1.V1ServerlessEndpoint, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	labels, err := validateLabels(opts.Labels)

	if err != nil {
		return nil, err
	}

	kind := sqlcv1.V1ServerlessEndpointKindCLOUDFLAREWORKERS

	if opts.Kind != nil {
		kind = *opts.Kind
	}

	enabled := true

	if opts.Enabled != nil {
		enabled = *opts.Enabled
	}

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	// The tenant row is upserted first so the endpoint's shard is computed against the
	// tenant's current shard_count within the same transaction.
	tenant, err := r.queries.UpsertServerlessTenant(ctx, tx, tenantId)

	if err != nil {
		return nil, fmt.Errorf("could not upsert serverless tenant: %w", err)
	}

	endpoint, err := r.queries.CreateServerlessEndpoint(ctx, tx, sqlcv1.CreateServerlessEndpointParams{
		ID:                    uuid.New(),
		Tenantid:              tenantId,
		Name:                  opts.Name,
		Kind:                  kind,
		Healthcheckurl:        opts.HealthcheckUrl,
		Triggerurl:            opts.TriggerUrl,
		Signingsecretenc:      opts.SigningSecretEnc,
		Requesttimeoutseconds: int32OrDefault(opts.RequestTimeoutSeconds, defaultServerlessRequestTimeoutSeconds),
		Pollintervalseconds:   int32OrDefault(opts.PollIntervalSeconds, defaultServerlessPollIntervalSeconds),
		Inlinewaitbudgetms:    int32OrDefault(opts.InlineWaitBudgetMs, defaultServerlessInlineWaitBudgetMs),
		Labels:                labels,
		Enabled:               enabled,
		Shardcount:            tenant.ShardCount,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create serverless endpoint: %w", err)
	}

	unit := sqlcv1.InsertServerlessLeaseIfAbsentParams{
		Tenantid: tenantId,
		Shard:    endpoint.Shard,
	}

	if err := r.queries.InsertServerlessLeaseIfAbsent(ctx, tx, unit); err != nil {
		return nil, fmt.Errorf("could not create serverless lease unit: %w", err)
	}

	err = r.queries.IncrementServerlessLeaseEndpointCount(ctx, tx, sqlcv1.IncrementServerlessLeaseEndpointCountParams{
		Tenantid: tenantId,
		Shard:    endpoint.Shard,
		Delta:    1,
	})

	if err != nil {
		return nil, fmt.Errorf("could not increment serverless lease endpoint count: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit serverless endpoint creation: %w", err)
	}

	return endpoint, nil
}

func (r *serverlessEndpointRepository) Get(ctx context.Context, tenantId, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error) {
	return r.queries.GetServerlessEndpoint(ctx, r.pool, sqlcv1.GetServerlessEndpointParams{
		Tenantid: tenantId,
		ID:       endpointId,
	})
}

func (r *serverlessEndpointRepository) GetById(ctx context.Context, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error) {
	return r.queries.GetServerlessEndpointById(ctx, r.pool, endpointId)
}

func (r *serverlessEndpointRepository) List(ctx context.Context, tenantId uuid.UUID, opts ListServerlessEndpointsOpts) ([]*sqlcv1.V1ServerlessEndpoint, int64, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, 0, err
	}

	endpoints, err := r.queries.ListServerlessEndpoints(ctx, r.pool, sqlcv1.ListServerlessEndpointsParams{
		Tenantid:       tenantId,
		Endpointlimit:  opts.Limit,
		Endpointoffset: opts.Offset,
	})

	if err != nil {
		return nil, 0, err
	}

	count, err := r.queries.CountServerlessEndpoints(ctx, r.pool, tenantId)

	if err != nil {
		return nil, 0, err
	}

	return endpoints, count, nil
}

func (r *serverlessEndpointRepository) Update(ctx context.Context, tenantId, endpointId uuid.UUID, opts UpdateServerlessEndpointOpts) (*sqlcv1.V1ServerlessEndpoint, error) {
	if err := r.v.Validate(opts); err != nil {
		return nil, err
	}

	params := sqlcv1.UpdateServerlessEndpointParams{
		Tenantid:              tenantId,
		ID:                    endpointId,
		Name:                  sqlchelpers.TextFromMaybeStr(opts.Name),
		HealthcheckUrl:        sqlchelpers.TextFromMaybeStr(opts.HealthcheckUrl),
		TriggerUrl:            sqlchelpers.TextFromMaybeStr(opts.TriggerUrl),
		SigningSecretEnc:      sqlchelpers.TextFromMaybeStr(opts.SigningSecretEnc),
		RequestTimeoutSeconds: sqlchelpers.ToInt(opts.RequestTimeoutSeconds),
		PollIntervalSeconds:   sqlchelpers.ToInt(opts.PollIntervalSeconds),
		InlineWaitBudgetMs:    sqlchelpers.ToInt(opts.InlineWaitBudgetMs),
	}

	if opts.Kind != nil {
		params.Kind = sqlcv1.NullV1ServerlessEndpointKind{
			V1ServerlessEndpointKind: *opts.Kind,
			Valid:                    true,
		}
	}

	if opts.Labels != nil {
		labels, err := validateLabels(opts.Labels)

		if err != nil {
			return nil, err
		}

		params.Labels = labels
	}

	if opts.Enabled != nil {
		params.Enabled = pgtype.Bool{Bool: *opts.Enabled, Valid: true}
	}

	return r.queries.UpdateServerlessEndpoint(ctx, r.pool, params)
}

func (r *serverlessEndpointRepository) Delete(ctx context.Context, tenantId, endpointId uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error) {
	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, r.pool, r.l)

	if err != nil {
		return nil, err
	}

	defer rollback()

	endpoint, err := r.queries.DeleteServerlessEndpoint(ctx, tx, sqlcv1.DeleteServerlessEndpointParams{
		Tenantid: tenantId,
		ID:       endpointId,
	})

	if err != nil {
		return nil, err
	}

	err = r.queries.IncrementServerlessLeaseEndpointCount(ctx, tx, sqlcv1.IncrementServerlessLeaseEndpointCountParams{
		Tenantid: tenantId,
		Shard:    endpoint.Shard,
		Delta:    -1,
	})

	if err != nil {
		return nil, fmt.Errorf("could not decrement serverless lease endpoint count: %w", err)
	}

	if err := commit(ctx); err != nil {
		return nil, fmt.Errorf("could not commit serverless endpoint deletion: %w", err)
	}

	return endpoint, nil
}

func (r *serverlessEndpointRepository) ListForUnits(ctx context.Context, units []ServerlessUnit, afterId uuid.UUID, limit int64) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	if len(units) == 0 {
		return nil, nil
	}

	tenantIds, shards := unitArrays(units)

	return r.queries.ListServerlessEndpointsForUnits(ctx, r.pool, sqlcv1.ListServerlessEndpointsForUnitsParams{
		Tenantids:     tenantIds,
		Shards:        shards,
		Afterid:       afterId,
		Endpointlimit: limit,
	})
}

func (r *serverlessEndpointRepository) ListForTenant(ctx context.Context, tenantId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	return r.queries.ListServerlessEndpointsForTenant(ctx, r.pool, tenantId)
}

func (r *serverlessEndpointRepository) ListUpdatedSince(ctx context.Context, tenantId uuid.UUID, since time.Time, sinceId uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	return r.queries.ListServerlessEndpointsUpdatedSince(ctx, r.pool, sqlcv1.ListServerlessEndpointsUpdatedSinceParams{
		Tenantid: tenantId,
		// A zero watermark is a real time, not NULL: a NULL would make the keyset comparison
		// NULL and the query return nothing for a tenant the cache has never seen a row of.
		Since:   pgtype.Timestamptz{Time: since, Valid: true},
		Sinceid: sinceId,
	})
}

func (r *serverlessEndpointRepository) GetByNamespace(ctx context.Context, tenantId, namespace uuid.UUID) (*sqlcv1.V1ServerlessEndpoint, error) {
	return r.queries.GetServerlessEndpointByNamespace(ctx, r.pool, sqlcv1.GetServerlessEndpointByNamespaceParams{
		Tenantid:  tenantId,
		Namespace: namespace,
	})
}

func (r *serverlessEndpointRepository) ListVersions(ctx context.Context, tenantId uuid.UUID, after ServerlessEndpointVersion, limit int64) ([]ServerlessEndpointVersion, error) {
	rows, err := r.queries.ListServerlessEndpointVersions(ctx, r.pool, sqlcv1.ListServerlessEndpointVersionsParams{
		Tenantid: tenantId,
		// The first page's zero version is a real time, not NULL; see ListUpdatedSince.
		Afterversion: pgtype.Timestamptz{Time: after.Version, Valid: true},
		Afterid:      after.ID,
		Versionlimit: limit,
	})

	if err != nil {
		return nil, err
	}

	out := make([]ServerlessEndpointVersion, 0, len(rows))

	for _, row := range rows {
		out = append(out, ServerlessEndpointVersion{ID: row.ID, Version: row.Version.Time})
	}

	return out, nil
}

func (r *serverlessEndpointRepository) ListByIds(ctx context.Context, ids []uuid.UUID) ([]*sqlcv1.V1ServerlessEndpoint, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	return r.queries.ListServerlessEndpointsByIds(ctx, r.pool, ids)
}

func (r *serverlessEndpointRepository) UpdateStatus(ctx context.Context, endpointId uuid.UUID, healthy bool, statusError *string) (time.Time, error) {
	changedAt, err := r.queries.UpdateServerlessEndpointStatus(ctx, r.pool, sqlcv1.UpdateServerlessEndpointStatusParams{
		ID:          endpointId,
		Healthy:     healthy,
		StatusError: sqlchelpers.TextFromMaybeStr(statusError),
	})

	if err != nil {
		return time.Time{}, err
	}

	return changedAt.Time, nil
}

func (r *serverlessEndpointRepository) UpdateRegisteredActions(ctx context.Context, endpointId uuid.UUID, actions []string) error {
	if actions == nil {
		actions = []string{}
	}

	return r.queries.UpdateServerlessEndpointRegisteredActions(ctx, r.pool, sqlcv1.UpdateServerlessEndpointRegisteredActionsParams{
		ID:                endpointId,
		Registeredactions: actions,
	})
}
