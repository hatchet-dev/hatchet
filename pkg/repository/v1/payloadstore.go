package v1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/telemetry"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog"
)

type StorePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
	Payload    []byte
}

type RetrievePayloadOpts struct {
	Id         int64
	InsertedAt pgtype.Timestamptz
	Type       sqlcv1.V1PayloadType
}

type PayloadLocation string
type ExternalPayloadLocationKey string

const (
	PayloadLocationInline   PayloadLocation = "inline"
	PayloadLocationExternal PayloadLocation = "external"
)

type PayloadContent struct {
	Location            PayloadLocation             `json:"location"`
	ExternalLocationKey *ExternalPayloadLocationKey `json:"external_location_key,omitempty"`
	InlineContent       []byte                      `json:"inline_content,omitempty"`
}

type ExternalStore interface {
	Store(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error)
	BulkRetrieve(ctx context.Context, tenantId string, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error)
}

type PayloadStoreOption func(*payloadStoreRepositoryImpl)

func WithExternalStore(store ExternalStore, externalStoreLocationName string, nativeStoreTTL time.Duration) PayloadStoreOption {
	return func(p *payloadStoreRepositoryImpl) {
		p.externalStoreEnabled = true
		p.externalStoreLocationName = &externalStoreLocationName
		p.nativeStoreTTL = &nativeStoreTTL
		p.externalStore = store
	}
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx sqlcv1.DBTX, tenantId string, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tenantId string, opts RetrievePayloadOpts) ([]byte, error)
	BulkRetrieve(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	ProcessPayloadWAL(ctx context.Context, tenantId string) (bool, error)
}

type payloadStoreRepositoryImpl struct {
	pool                      *pgxpool.Pool
	l                         *zerolog.Logger
	queries                   *sqlcv1.Queries
	externalStoreEnabled      bool
	externalStoreLocationName *string
	nativeStoreTTL            *time.Duration
	externalStore             ExternalStore
}

func NewPayloadStoreRepository(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	queries *sqlcv1.Queries,
	opts ...PayloadStoreOption,
) PayloadStoreRepository {
	repo := &payloadStoreRepositoryImpl{
		pool:                      pool,
		l:                         l,
		queries:                   queries,
		externalStoreEnabled:      false,
		externalStoreLocationName: nil,
		nativeStoreTTL:            nil,
		externalStore:             &NoOpExternalStore{},
	}

	for _, opt := range opts {
		opt(repo)
	}

	return repo
}

func (p PayloadContent) Validate() error {
	switch p.Location {
	case PayloadLocationInline:
		if len(p.InlineContent) == 0 {
			return fmt.Errorf("inline content cannot be empty when location is %s", PayloadLocationInline)
		}

		if p.ExternalLocationKey != nil {
			return fmt.Errorf("external location key must be nil when location is %s", PayloadLocationInline)
		}

		return nil
	case PayloadLocationExternal:
		if p.ExternalLocationKey == nil || len(*p.ExternalLocationKey) == 0 {
			return fmt.Errorf("external location key cannot be empty when location is %s", PayloadLocationExternal)
		}

		if len(p.InlineContent) > 0 {
			return fmt.Errorf("inline content must be empty when location is %s", PayloadLocationExternal)
		}

		return nil
	default:
		return fmt.Errorf("invalid payload location: %s", p.Location)
	}
}

func (p PayloadContent) Marshal() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, fmt.Errorf("invalid payload: %w", err)
	}

	j, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return j, nil
}

func UnmarshalPayloadContent(data []byte) (*PayloadContent, error) {
	var content PayloadContent
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload content: %w", err)
	}

	if err := content.Validate(); err != nil {
		return nil, fmt.Errorf("invalid payload content: %w", err)
	}

	return &content, nil
}

func (p *payloadStoreRepositoryImpl) generateExternalKey(tenantId string, payload StorePayloadOpts) string {
	// TODO: Not sure if I need a key generator like this
	return fmt.Sprintf("%s/%s/%d/%d", tenantId, payload.Type, payload.Id, payload.InsertedAt.Time.Unix())
}

func (p *payloadStoreRepositoryImpl) shouldOffloadToExternal(payload []byte) bool {
	// TODO - need to add some logic here based on TTL
	return p.externalStoreEnabled
}

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx sqlcv1.DBTX, tenantId string, payloads ...StorePayloadOpts) error {
	taskIds := make([]int64, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, len(payloads))
	payloadTypes := make([]string, len(payloads))
	payloadData := make([][]byte, len(payloads))
	offloadAts := make([]pgtype.Timestamptz, len(payloads))
	operations := make([]string, len(payloads))

	for i, payload := range payloads {
		taskIds[i] = payload.Id
		taskInsertedAts[i] = payload.InsertedAt
		payloadTypes[i] = string(payload.Type)

		if p.externalStoreEnabled {
			offloadAts[i] = pgtype.Timestamptz{Time: payload.InsertedAt.Time.Add(*p.nativeStoreTTL), Valid: true}
		} else {
			offloadAts[i] = pgtype.Timestamptz{Time: time.Now(), Valid: true}
		}

		operations[i] = string(sqlcv1.V1PayloadWalOperationINSERT)

		// Always store inline initially - offloading happens via cron job
		content := PayloadContent{
			Location:      PayloadLocationInline,
			InlineContent: payload.Payload,
		}
		marshaledContent, err := content.Marshal()
		if err != nil {
			return fmt.Errorf("failed to marshal inline payload: %w", err)
		}
		payloadData[i] = marshaledContent
	}

	err := p.queries.WritePayloads(ctx, p.pool, sqlcv1.WritePayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
		Payloads:    payloadData,
	})

	if err != nil {
		return fmt.Errorf("failed to write payloads: %w", err)
	}

	err = p.queries.WritePayloadWAL(ctx, tx, sqlcv1.WritePayloadWALParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Payloadids:         taskIds,
		Payloadinsertedats: taskInsertedAts,
		Payloadtypes:       payloadTypes,
		Offloadats:         offloadAts,
		Operations:         operations,
	})

	if err != nil {
		return fmt.Errorf("failed to write payload WAL: %w", err)
	}

	return err
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tenantId string, opts RetrievePayloadOpts) ([]byte, error) {
	payloadMap, err := p.BulkRetrieve(ctx, tenantId, opts)

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	payload, ok := payloadMap[opts]

	if !ok {
		return nil, fmt.Errorf("no payload found for opts: %+v", opts)
	}

	return payload, nil
}

func (p *payloadStoreRepositoryImpl) BulkRetrieve(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	return p.bulkRetrieve(ctx, p.pool, tenantId, opts...)
}

func (p *payloadStoreRepositoryImpl) bulkRetrieve(ctx context.Context, tx sqlcv1.DBTX, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if len(opts) == 0 {
		return make(map[RetrievePayloadOpts][]byte), nil
	}

	taskIds := make([]int64, len(opts))
	taskInsertedAts := make([]pgtype.Timestamptz, len(opts))
	payloadTypes := make([]string, len(opts))

	for i, opt := range opts {
		taskIds[i] = opt.Id
		taskInsertedAts[i] = opt.InsertedAt
		payloadTypes[i] = string(opt.Type)
	}

	payloads, err := p.queries.ReadPayloads(ctx, tx, sqlcv1.ReadPayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)

	externalKeysToOpts := make(map[ExternalPayloadLocationKey]RetrievePayloadOpts)
	externalKeys := make([]ExternalPayloadLocationKey, 0)

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		content, err := UnmarshalPayloadContent(payload.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload content: %w", err)
		}

		opts := RetrievePayloadOpts{
			Id:         payload.ID,
			InsertedAt: payload.InsertedAt,
			Type:       payload.Type,
		}

		if content.Location == PayloadLocationExternal && content.ExternalLocationKey != nil {
			externalKeysToOpts[*content.ExternalLocationKey] = opts
			externalKeys = append(externalKeys, *content.ExternalLocationKey)
		} else {
			optsToPayload[opts] = content.InlineContent
		}
	}

	if len(externalKeys) > 0 {
		externalData, err := p.externalStore.BulkRetrieve(ctx, tenantId, externalKeys...)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve external payloads: %w", err)
		}

		for externalKey, data := range externalData {
			if opt, exists := externalKeysToOpts[externalKey]; exists {
				optsToPayload[opt] = data
			}
		}
	}

	return optsToPayload, nil
}

func (p *payloadStoreRepositoryImpl) offloadToExternal(ctx context.Context, tx sqlcv1.DBTX, tenantId string, payloads ...StorePayloadOpts) error {
	if !p.externalStoreEnabled {
		return nil
	}

	retrieveOpts, err := p.externalStore.Store(ctx, tenantId, payloads...)

	if err != nil {
		return fmt.Errorf("failed to store payloads externally: %w", err)
	}

	ids := make([]int64, len(retrieveOpts))
	insertedAts := make([]pgtype.Timestamptz, len(retrieveOpts))
	payloadsToOffload := make([][]byte, len(retrieveOpts))

	for opt, key := range retrieveOpts {
		content := PayloadContent{
			Location:            PayloadLocationExternal,
			ExternalLocationKey: &key,
		}

		marshaledContent, err := content.Marshal()

		if err != nil {
			return fmt.Errorf("failed to marshal external payload content: %w", err)
		}

		ids = append(ids, opt.Id)
		insertedAts = append(insertedAts, opt.InsertedAt)
		payloadsToOffload = append(payloadsToOffload, marshaledContent)
	}

	// TODO: Update payloads in the db in a tx-safe way here?
	return p.queries.OffloadPayloadsToExternalStore(ctx, tx, sqlcv1.OffloadPayloadsToExternalStoreParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         ids,
		Insertedats: insertedAts,
		Values:      payloadsToOffload,
	})
}

func (p *payloadStoreRepositoryImpl) ProcessPayloadWAL(ctx context.Context, tenantId string) (bool, error) {
	ctx, span := telemetry.NewSpan(ctx, "process-payload-wal")
	defer span.End()

	tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, p.pool, p.l, 5000)

	defer rollback()

	if err != nil {
		return false, fmt.Errorf("failed to prepare transaction: %w", err)
	}

	walRecords, err := p.queries.PollPayloadWALForRecordsToOffload(ctx, tx, sqlcv1.PollPayloadWALForRecordsToOffloadParams{
		Tenantid:  sqlchelpers.UUIDFromStr(tenantId),
		Polllimit: 1000,
	})

	if len(walRecords) == 0 {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	retrieveOpts := make([]RetrievePayloadOpts, len(walRecords))
	offloadAts := make([]pgtype.Timestamptz, len(walRecords))
	payloadIds := make([]int64, len(walRecords))
	payloadInsertedAts := make([]pgtype.Timestamptz, len(walRecords))
	payloadTypes := make([]string, len(walRecords))

	for i, record := range walRecords {
		retrieveOpts[i] = RetrievePayloadOpts{
			Id:         record.PayloadID,
			InsertedAt: record.PayloadInsertedAt,
			Type:       record.PayloadType,
		}

		offloadAts[i] = record.OffloadAt
		payloadIds[i] = record.PayloadID
		payloadInsertedAts[i] = record.PayloadInsertedAt
		payloadTypes[i] = string(record.PayloadType)
	}

	payloads, err := p.bulkRetrieve(ctx, tx, tenantId, retrieveOpts...)

	if err != nil {
		return false, err
	}

	externalStoreOpts := make([]StorePayloadOpts, 0)

	for opts, payload := range payloads {
		externalStoreOpts = append(externalStoreOpts, StorePayloadOpts{
			Id:         opts.Id,
			InsertedAt: opts.InsertedAt,
			Type:       opts.Type,
			Payload:    payload,
		})
	}

	err = p.offloadToExternal(ctx, tx, tenantId, externalStoreOpts...)

	if err != nil {
		return false, err
	}

	err = p.queries.EvictPayloadWALRecords(ctx, tx, sqlcv1.EvictPayloadWALRecordsParams{
		Tenantid:           sqlchelpers.UUIDFromStr(tenantId),
		Payloadids:         payloadIds,
		Payloadinsertedats: payloadInsertedAts,
		Offloadats:         offloadAts,
		Payloadtypes:       payloadTypes,
	})

	if err != nil {
		return false, err
	}

	if err := commit(ctx); err != nil {
		return false, err
	}

	return true, nil
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]ExternalPayloadLocationKey, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) BulkRetrieve(ctx context.Context, tenantId string, keys ...ExternalPayloadLocationKey) (map[ExternalPayloadLocationKey][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
