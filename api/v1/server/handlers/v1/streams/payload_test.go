package streams

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/o11yusage"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type entitledTenants struct {
	repository.TenantEntitlementRepository
}

func (entitledTenants) HasEntitlement(context.Context, uuid.UUID, repository.Entitlement) (bool, error) {
	return true, nil
}

type storedPayloads struct {
	repository.StreamsRepository

	payloads map[uuid.UUID][]byte
}

func (s storedPayloads) GetStreamPayload(_ context.Context, _ uuid.UUID, ref repository.StreamPayloadRef) ([]byte, error) {
	payload, ok := s.payloads[ref.ID]

	if !ok {
		return nil, repository.ErrStreamPayloadNotFound
	}

	return payload, nil
}

type payloadRepository struct {
	repository.Repository

	streams storedPayloads
}

func (r payloadRepository) TenantEntitlement() repository.TenantEntitlementRepository {
	return entitledTenants{}
}

func (r payloadRepository) Streams() repository.StreamsRepository { return r.streams }

func TestPayloadGetBillsTheBytesItReturns(t *testing.T) {
	tenantId := uuid.New()
	stored := repository.StreamPayloadRef{ID: uuid.New(), CreatedAt: time.Now()}
	missing := repository.StreamPayloadRef{ID: uuid.New(), CreatedAt: time.Now()}

	var mu sync.Mutex
	delivered := int64(0)
	l := zerolog.Nop()
	usage := o11yusage.NewAggregator(&l, time.Hour, func(tenants map[uuid.UUID]o11yusage.TenantBytes) error {
		mu.Lock()
		defer mu.Unlock()
		delivered += tenants[tenantId].StreamDelivered
		return nil
	})
	usage.Start()

	s := NewV1StreamsService(&server.ServerConfig{
		Layer: &database.Layer{V1: payloadRepository{
			streams: storedPayloads{payloads: map[uuid.UUID][]byte{stored.ID: make([]byte, 1000)}},
		}},
	}, usage)

	get := func(ref repository.StreamPayloadRef, member *sqlcv1.PopulateTenantMembersRow, w http.ResponseWriter) error {
		t.Helper()

		encoded, err := repository.EncodeStreamPayloadRef(ref)
		require.NoError(t, err)

		ctx := echo.New().NewContext(httptest.NewRequest(http.MethodGet, "/", nil), httptest.NewRecorder())
		ctx.Set("tenant", &sqlcv1.Tenant{ID: tenantId})

		if member != nil {
			ctx.Set("tenant-member", member)
		}

		resp, err := s.V1StreamPayloadGet(ctx, gen.V1StreamPayloadGetRequestObject{Params: gen.V1StreamPayloadGetParams{Ref: encoded}})
		require.NoError(t, err)

		return resp.VisitV1StreamPayloadGetResponse(w)
	}

	downloaded := httptest.NewRecorder()
	require.NoError(t, get(stored, nil, downloaded))
	assert.Equal(t, http.StatusOK, downloaded.Code)
	assert.Len(t, downloaded.Body.Bytes(), 1000)

	require.NoError(t, get(stored, nil, httptest.NewRecorder()))
	require.ErrorIs(t, get(stored, nil, &failingWriter{ResponseRecorder: httptest.NewRecorder(), accept: 400}), errClientGone)

	notFound := httptest.NewRecorder()
	require.NoError(t, get(missing, nil, notFound))
	assert.Equal(t, http.StatusNotFound, notFound.Code)

	forbidden := httptest.NewRecorder()
	require.NoError(t, get(stored, &sqlcv1.PopulateTenantMembersRow{Role: sqlcv1.TenantMemberRoleMEMBER}, forbidden))
	assert.Equal(t, http.StatusForbidden, forbidden.Code)

	require.NoError(t, usage.Shutdown())

	mu.Lock()
	defer mu.Unlock()
	assert.Equal(t, int64(2000), delivered, "each download written in full, at its size; a download cut off partway, refused or missing not at all")
}

var errClientGone = errors.New("client went away")

// failingWriter takes the first accept bytes of the body, then fails as a disconnected client would.
type failingWriter struct {
	*httptest.ResponseRecorder

	accept int
}

func (w *failingWriter) Write(p []byte) (int, error) {
	if len(p) <= w.accept {
		w.accept -= len(p)
		return w.ResponseRecorder.Write(p)
	}

	n, _ := w.ResponseRecorder.Write(p[:w.accept])
	w.accept = 0

	return n, errClientGone
}
