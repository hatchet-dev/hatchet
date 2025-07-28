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

type ExternalHandler interface {
	OffloadToExternal(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) error
	RetrieveFromExternal(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
}

type PayloadStoreRepository interface {
	Store(ctx context.Context, tx pgx.Tx, tenantId string, payloads ...StorePayloadOpts) error
	Retrieve(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error)
	ExternalHandler
}

type payloadStoreRepositoryImpl struct {
	pool                      *pgxpool.Pool
	l                         *zerolog.Logger
	queries                   *sqlcv1.Queries
	externalStoreEnabled      bool
	externalStoreLocationName *string
	nativeStoreTTL            *int64
	externalHandler           ExternalHandler
}

func NewPayloadStoreRepository(
	pool *pgxpool.Pool,
	l *zerolog.Logger,
	queries *sqlcv1.Queries,
	externalStoreEnabled bool,
	externalStoreLocationName *string,
	nativeStoreTTL *int64,
	externalHandler ExternalHandler,
) PayloadStoreRepository {
	if externalStoreEnabled && nativeStoreTTL == nil {
		panic("nativeStoreTTL must be set when externalStoreEnabled is true")
	}

	return &payloadStoreRepositoryImpl{
		pool:                      pool,
		l:                         l,
		queries:                   queries,
		externalStoreEnabled:      externalStoreEnabled,
		externalStoreLocationName: externalStoreLocationName,
		nativeStoreTTL:            nativeStoreTTL,
		externalHandler:           externalHandler,
	}
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
		payloadData[i] = payload.Payload
	}

	return p.queries.WritePayloads(ctx, tx, sqlcv1.WritePayloadsParams{
		Tenantid:    sqlchelpers.UUIDFromStr(tenantId),
		Ids:         taskIds,
		Insertedats: taskInsertedAts,
		Types:       payloadTypes,
		Payloads:    payloadData,
	})
}

func (p *payloadStoreRepositoryImpl) Retrieve(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
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
	externalOptsToRetrieve := make([]RetrievePayloadOpts, 0)

	for _, payload := range payloads {
		if payload == nil {
			continue
		}

		content, err := UnmarshalPayloadContent(payload.Value)
		if err != nil {
			p.l.Error().Err(err).Msg("failed to unmarshal payload content")
			continue
		}

		opts := RetrievePayloadOpts{
			Id:         payload.ID,
			InsertedAt: payload.InsertedAt,
			Type:       payload.Type,
		}

		if content.Location == PayloadLocationExternal && content.ExternalLocationKey != nil {
			externalOptsToRetrieve = append(externalOptsToRetrieve, opts)
		} else {
			optsToPayload[opts] = content.InlineContent
		}
	}

	data, err := p.externalHandler.RetrieveFromExternal(ctx, tenantId, externalOptsToRetrieve...)

	for opt, content := range data {
		if content == nil {
			p.l.Warn().Interface("opt", opt).Msg("external payload content is nil")
			continue
		}
		optsToPayload[opt] = content
	}

	return optsToPayload, nil
}

func (p *payloadStoreRepositoryImpl) OffloadToExternal(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) error {
	if p.externalHandler == nil {
		return fmt.Errorf("no external handler configured")
	}

	return p.externalHandler.OffloadToExternal(ctx, tenantId, payloads...)
}

func (p *payloadStoreRepositoryImpl) RetrieveFromExternal(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	if p.externalHandler == nil {
		return nil, fmt.Errorf("no external handler configured")
	}

	return p.externalHandler.RetrieveFromExternal(ctx, tenantId, opts...)
}

type DefaultExternalHandler struct {
}

func NewDefaultExternalHandler() ExternalHandler {
	return &DefaultExternalHandler{}
}

func (d *DefaultExternalHandler) OffloadToExternal(ctx context.Context, tenantId string, payloads ...StorePayloadOpts) error {
	return nil
}

func (d *DefaultExternalHandler) RetrieveFromExternal(ctx context.Context, tenantId string, opts ...RetrievePayloadOpts) (map[RetrievePayloadOpts][]byte, error) {
	return nil, fmt.Errorf("retrieve from external not implemented")
}
