package streams

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

func encodedCursor(t *testing.T, c v1.StreamCursor) string {
	t.Helper()

	s, err := v1.EncodeStreamCursor(c)
	require.NoError(t, err)

	return s
}

type streamsOnlyRepository struct {
	v1.Repository
	streams v1.StreamsRepository
}

func (r *streamsOnlyRepository) Streams() v1.StreamsRepository {
	return r.streams
}

func TestResolveCursor(t *testing.T) {
	tenantId := uuid.New()
	ctx := context.Background()

	repo := newFakeStreamsRepository(tenantId, "ns-a", "topic-a", 3, time.Now())
	repo.expiredThroughID = 2
	s := &ServiceImpl{repo: &streamsOnlyRepository{streams: repo}}

	tooLong := strings.Repeat("a", v1.MaxStreamNameLength+1)
	ownCursor := encodedCursor(t, v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now(), ID: 7})
	// a cursor is client-supplied, so the address decoded from it is checked too
	tooLongCursor := encodedCursor(t, v1.StreamCursor{Namespace: tooLong, Topic: "orders", ID: 1})
	expiredCursor := encodedCursor(t, v1.StreamCursor{Namespace: "ns-a", Topic: "topic-a", CreatedAt: time.Now(), ID: 1})

	t.Run("a cursor for the topic resolves", func(t *testing.T) {
		cursor, err := s.resolveCursor(ctx, tenantId, "ns-a", "topic-a", ownCursor)
		require.NoError(t, err)
		assert.Equal(t, int64(7), cursor.ID)
	})

	t.Run("no topic takes the cursor's own", func(t *testing.T) {
		cursor, err := s.resolveCursor(ctx, tenantId, "", "", ownCursor)
		require.NoError(t, err)
		assert.Equal(t, "ns-a", cursor.Namespace)
		assert.Equal(t, "topic-a", cursor.Topic)
	})

	rejected := map[string]struct {
		namespace, topic, cursor string
		wantCode                 connect.Code
		wantErr                  string
	}{
		"a malformed cursor":                {topic: "topic-a", cursor: "nope", wantCode: connect.CodeInvalidArgument},
		"a cursor from another topic":       {namespace: "ns-a", topic: "topic-b", cursor: ownCursor, wantCode: connect.CodeInvalidArgument},
		"a cursor from another namespace":   {namespace: "ns-b", topic: "topic-a", cursor: ownCursor, wantCode: connect.CodeInvalidArgument},
		"a cursor whose address is invalid": {cursor: tooLongCursor, wantCode: connect.CodeInvalidArgument, wantErr: "namespace is 256 characters long"},
		"an expired cursor":                 {namespace: "ns-a", topic: "topic-a", cursor: expiredCursor, wantCode: connect.CodeOutOfRange},
	}

	for label, tc := range rejected {
		t.Run("rejects "+label, func(t *testing.T) {
			_, err := s.resolveCursor(ctx, tenantId, tc.namespace, tc.topic, tc.cursor)
			assert.Equal(t, tc.wantCode, connect.CodeOf(err))

			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			}
		})
	}
}

func TestSubscribeRejectsInvalidAddress(t *testing.T) {
	tenantId := uuid.New()

	s := &ServiceImpl{
		repo: &entitlementRepository{entitlements: &fakeEntitlements{entitled: map[uuid.UUID]bool{tenantId: true}}},
	}

	ctx := context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: tenantId}) // nolint:staticcheck

	rejected := map[string]struct {
		req     *contracts.SubscribeStreamRequest
		wantErr string
	}{
		"no topic and no cursor":  {req: &contracts.SubscribeStreamRequest{}, wantErr: "topic is required"},
		"a topic that's too long": {req: &contracts.SubscribeStreamRequest{Topic: strings.Repeat("a", v1.MaxStreamNameLength+1)}, wantErr: "topic is 256 characters long"},
	}

	for label, tc := range rejected {
		t.Run("rejects "+label, func(t *testing.T) {
			cs, err := s.RegisterSubscribe(ctx, tc.req)
			require.NoError(t, err)

			for range cs.Responses {
				t.Fatal("a rejected subscription sends nothing")
			}

			assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(cs.Err))
			assert.ErrorContains(t, cs.Err, tc.wantErr)
		})
	}
}
