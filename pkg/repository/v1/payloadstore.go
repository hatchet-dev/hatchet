package v1

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
	"github.com/jackc/pgx/v5"
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

const (
	PayloadLocationInline   PayloadLocation = "inline"
	PayloadLocationExternal PayloadLocation = "external"
)

type PayloadContent struct {
	Location            PayloadLocation `json:"location"`
	ExternalLocationKey *string         `json:"external_location_key,omitempty"`
	InlineContent       []byte          `json:"inline_content,omitempty"`
}

type ExternalStore interface {
	Store(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]string, error)
	BulkRetrieve(ctx context.Context, tenantId string, keys ...string) (map[string][]byte, error)
}

type PayloadStoreOption func(*payloadStoreRepositoryImpl)

func WithExternalStore(store ExternalStore, externalStoreLocationName string, nativeStoreTTL int64) PayloadStoreOption {
	return func(p *payloadStoreRepositoryImpl) {
		p.externalStoreEnabled = true
		p.externalStoreLocationName = &externalStoreLocationName
		p.nativeStoreTTL = &nativeStoreTTL
		p.externalStore = store
	}
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx pgx.Tx, tenantId string, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tenantId string, opts RetrievePayloadOpts) ([]byte, error)
	BulkRetrieve(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	OffloadToExternal(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) error
}

type payloadStoreRepositoryImpl struct {
	pool                      *pgxpool.Pool
	l                         *zerolog.Logger
	queries                   *sqlcv1.Queries
	externalStoreEnabled      bool
	externalStoreLocationName *string
	nativeStoreTTL            *int64
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

func (p *payloadStoreRepositoryImpl) Store(ctx context.Context, tx pgx.Tx, tenantId string, payloads ...StorePayloadOpts) error {
	taskIds := make([]int64, len(payloads))
	taskInsertedAts := make([]pgtype.Timestamptz, len(payloads))
	payloadTypes := make([]string, len(payloads))
	payloadData := make([][]byte, len(payloads))

	for i, payload := range payloads {
		taskIds[i] = payload.Id
		taskInsertedAts[i] = payload.InsertedAt
		payloadTypes[i] = string(payload.Type)

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

	return p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
		Payloads:    payloadData,
	})
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

	payloads, err := p.queries.ReadPayloads(ctx, p.pool, sqlcv1.ReadPayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to read payload metadata: %w", err)
	}

	optsToPayload := make(map[RetrievePayloadOpts][]byte)

	externalKeysToOpts := make(map[string]RetrievePayloadOpts)
	externalKeys := make([]string, 0)

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

func (p *payloadStoreRepositoryImpl) OffloadToExternal(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) error {
	if !p.externalStoreEnabled {
		return fmt.Errorf("external store not enabled")
	}

	_, err := p.externalStore.Store(ctx, tenantId, payloads...)

	// TODO: Update payloads in the db in a tx-safe way here?

	if err != nil {
		return fmt.Errorf("failed to store payloads externally: %w", err)
	}

	return nil
}

type NoOpExternalStore struct{}

func (n *NoOpExternalStore) Store(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) (map[RetrievePayloadOpts]string, error) {
	return nil, fmt.Errorf("external store disabled")
}

func (n *NoOpExternalStore) BulkRetrieve(ctx context.Context, tenantId string, keys ...string) (map[string][]byte, error) {
	return nil, fmt.Errorf("external store disabled")
}
