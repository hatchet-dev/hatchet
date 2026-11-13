package streams

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	contracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/internal/services/shared/streams"
	"github.com/hatchet-dev/hatchet/pkg/analytics"
	"github.com/hatchet-dev/hatchet/pkg/enums"
	"github.com/hatchet-dev/hatchet/pkg/o11yusage"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// meteringStreamsRepository stores every message except a "duplicate", which it
// reports as already stored, and a "gap" and everything after it.
type meteringStreamsRepository struct {
	v1.StreamsRepository

	uploadedSizes map[uuid.UUID]int64
}

func (f *meteringStreamsRepository) EnsureTopic(context.Context, uuid.UUID, string, string) error {
	return nil
}

func (f *meteringStreamsRepository) StreamPayloadSize(_ context.Context, _ uuid.UUID, ref v1.StreamPayloadRef) (int64, error) {
	size, ok := f.uploadedSizes[ref.ID]

	if !ok {
		return 0, v1.ErrStreamPayloadNotFound
	}

	return size, nil
}

func (f *meteringStreamsRepository) InsertOrderedStreamMessages(_ context.Context, msgs []v1.TenantStreamMessage) ([]v1.OrderedStreamMessageResult, error) {
	results := make([]v1.OrderedStreamMessageResult, len(msgs))
	gap := false

	for i, m := range msgs {
		seq := m.Opts.ProducerSeq

		switch {
		case gap || string(m.Opts.Payload) == "gap":
			gap = true
			results[i] = v1.OrderedStreamMessageResult{CurrentSeq: seq - 2}
		case string(m.Opts.Payload) == "duplicate":
			results[i] = v1.OrderedStreamMessageResult{CurrentSeq: seq}
		default:
			results[i] = v1.OrderedStreamMessageResult{Inserted: true, CurrentSeq: seq - 1}
		}
	}

	return results, nil
}

type unlimitedTenants struct {
	v1.TenantLimitRepository
}

func (unlimitedTenants) Meter(context.Context, sqlcv1.DBTX, enums.LimitResource, uuid.UUID, int32) (func() error, func()) {
	return func() error { return nil }, func() {}
}

type meteringRepository struct {
	v1.Repository

	streams      *meteringStreamsRepository
	entitlements *fakeEntitlements
}

func (r *meteringRepository) Streams() v1.StreamsRepository { return r.streams }

func (r *meteringRepository) TenantEntitlement() v1.TenantEntitlementRepository {
	return r.entitlements
}

func (r *meteringRepository) TenantLimit() v1.TenantLimitRepository { return unlimitedTenants{} }

// recordingUsage returns an aggregator and a func that shuts it down and returns everything it flushed.
func recordingUsage(t *testing.T) (*o11yusage.Aggregator, func() map[uuid.UUID]o11yusage.TenantBytes) {
	t.Helper()

	var mu sync.Mutex
	flushed := map[uuid.UUID]o11yusage.TenantBytes{}

	usage := o11yusage.NewAggregator(testLogger(), time.Hour, func(tenants map[uuid.UUID]o11yusage.TenantBytes) error {
		mu.Lock()
		defer mu.Unlock()

		for id, b := range tenants {
			cur := flushed[id]
			cur.StreamPublished += b.StreamPublished
			cur.StreamDelivered += b.StreamDelivered
			flushed[id] = cur
		}

		return nil
	})
	usage.Start()

	return usage, func() map[uuid.UUID]o11yusage.TenantBytes {
		require.NoError(t, usage.Shutdown())

		mu.Lock()
		defer mu.Unlock()

		return flushed
	}
}

func TestPublishBillsTheStoredMessagesPayloadBytes(t *testing.T) {
	tenantId := uuid.New()
	uploaded := v1.StreamPayloadRef{ID: uuid.New(), CreatedAt: time.Now()}
	streams := &meteringStreamsRepository{uploadedSizes: map[uuid.UUID]int64{uploaded.ID: 1000}}
	usage, flushed := recordingUsage(t)

	recorded := &recordingAnalytics{}
	s := &ServiceImpl{
		repo: &meteringRepository{
			streams:      streams,
			entitlements: &fakeEntitlements{entitled: map[uuid.UUID]bool{tenantId: true}},
		},
		usage:     usage,
		analytics: recorded,
		publisher: newPublishBatcher(streams, fakePubSub{}, testLogger(), 1),
	}
	t.Cleanup(s.publisher.stop)

	ref, err := v1.EncodeStreamPayloadRef(uploaded)
	require.NoError(t, err)

	tokenID := uuid.New()
	ctx := context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: tenantId}) // nolint:staticcheck
	ctx = context.WithValue(ctx, analytics.APITokenIDKey, tokenID)
	ctx = context.WithValue(ctx, analytics.SourceKey, analytics.SourceGRPC)

	err = s.publish(ctx, "", "topic", "producer", 0, []*contracts.StreamPublishEntry{
		{Payload: []byte("inline")},
		{Payload: []byte("duplicate")},
		{PayloadRef: ref},
		{Payload: []byte("gap")},
		{Payload: []byte("after-gap")},
	})
	assert.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err), "the gap fails the publish after the messages ahead of it are stored")

	err = s.publish(ctx, "", "topic", "producer", 1, []*contracts.StreamPublishEntry{
		{Payload: []byte("duplicate")},
	})
	assert.NoError(t, err)

	assert.Equal(t, o11yusage.TenantBytes{StreamPublished: int64(len("inline")) + 1000}, flushed()[tenantId],
		"only stored messages are billed, an uploaded payload at its uploaded size")

	recorded.mu.Lock()
	defer recorded.mu.Unlock()
	require.Len(t, recorded.events, 1, "a retry of a stored message is not counted again")
	assert.Equal(t, recordedCount{
		resource: analytics.DurableStream,
		action:   analytics.Publish,
		tenant:   tenantId,
		source:   analytics.SourceGRPC,
		n:        2,
		bytes:    int64(len("inline")) + 1000,
	}, recorded.events[0], "only stored messages are counted, aggregated for the tenant rather than the token")
}

type recordedCount struct {
	resource analytics.Resource
	action   analytics.Action
	tenant   uuid.UUID
	source   analytics.Source
	n        int64
	bytes    int64
	hasToken bool
}

type recordingAnalytics struct {
	analytics.NoOpAnalytics

	mu     sync.Mutex
	events []recordedCount
}

func (r *recordingAnalytics) CountN(ctx context.Context, resource analytics.Resource, action analytics.Action, n int64, totals analytics.Totals, _ ...analytics.Properties) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var tenant uuid.UUID
	if id := analytics.TenantIDFromContext(ctx); id != nil {
		tenant = *id
	}

	r.events = append(r.events, recordedCount{
		resource: resource,
		action:   action,
		tenant:   tenant,
		source:   analytics.SourceFromContext(ctx),
		n:        n,
		bytes:    totals["bytes"],
		hasToken: analytics.TokenIDFromContext(ctx) != nil,
	})
}

type subscribableRepository struct {
	v1.Repository

	streams      *fakeStreamsRepository
	entitlements *fakeEntitlements
}

func (r *subscribableRepository) Streams() v1.StreamsRepository { return r.streams }

func (r *subscribableRepository) TenantEntitlement() v1.TenantEntitlementRepository {
	return r.entitlements
}

func TestInProcessSubscribersAreNotBilled(t *testing.T) {
	tenantId := uuid.New()
	streamsRepo := newFakeStreamsRepository(tenantId, "", "topic", 0, time.Now())
	usage, flushed := recordingUsage(t)

	s := &ServiceImpl{
		repo: &subscribableRepository{
			streams:      streamsRepo,
			entitlements: &fakeEntitlements{entitled: map[uuid.UUID]bool{tenantId: true}},
		},
		usage:          usage,
		streamSessions: streams.NewRegistry(),
		topicPollers:   newTopicPollerRegistry(streamsRepo, fakePubSub{}, testLogger(), 5*time.Millisecond, time.Hour),
	}
	t.Cleanup(func() { require.NoError(t, s.topicPollers.Close()) })

	ctx, cancel := context.WithCancel(context.WithValue(context.Background(), "tenant", &sqlcv1.Tenant{ID: tenantId})) // nolint:staticcheck
	defer cancel()

	cs, err := s.RegisterSubscribe(ctx, &contracts.SubscribeStreamRequest{Topic: "topic"})
	require.NoError(t, err)

	streamsRepo.appendTopicMessage("topic", 1)

	select {
	case resp := <-cs.Responses:
		msg := resp.(*contracts.StreamMessage)
		require.Len(t, msg.Entries, 1)
		assert.Equal(t, []byte("topic"), msg.Entries[0].Payload)
	case <-time.After(time.Second):
		t.Fatal("the in-process subscriber never received the message")
	}

	cancel()

	for range cs.Responses {
	}

	assert.NotContains(t, flushed(), tenantId, "engine-internal subscribers aren't billed for what they read")
}

func TestMeteredSendBillsTheInlinePayloadBytesItDelivers(t *testing.T) {
	tenantId := uuid.New()
	usage, flushed := recordingUsage(t)
	s := &ServiceImpl{usage: usage}

	errSend := errors.New("send failed")
	delivered := s.meteredSend(tenantId, func(*contracts.StreamMessage) error { return nil })
	failed := s.meteredSend(tenantId, func(*contracts.StreamMessage) error { return errSend })

	frame := &contracts.StreamMessage{Entries: []*contracts.StreamEntry{
		{Payload: []byte("abc")},
		{Payload: []byte("de")},
		{PayloadRef: "fetched separately"},
	}}

	require.NoError(t, delivered(frame))
	require.NoError(t, delivered(frame))
	require.ErrorIs(t, failed(frame), errSend)
	require.NoError(t, delivered(&contracts.StreamMessage{Hangup: true}))

	assert.Equal(t, o11yusage.TenantBytes{StreamDelivered: 10}, flushed()[tenantId],
		"each delivered frame's inline payloads; a ref is billed when it's fetched, and a failed send not at all")
}
