package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type fakeV1StreamsClient struct {
	sharedcontracts.V1StreamsClient

	errs []error
	reqs []*sharedcontracts.PublishStreamMessageRequest

	metadata map[string]*sharedcontracts.StreamTopicMetadata
}

func (f *fakeV1StreamsClient) Publish(_ context.Context, req *sharedcontracts.PublishStreamMessageRequest, _ ...grpc.CallOption) (*sharedcontracts.PublishStreamMessageResponse, error) {
	f.reqs = append(f.reqs, req)

	var err error
	if len(f.errs) > 0 {
		err, f.errs = f.errs[0], f.errs[1:]
	}

	return &sharedcontracts.PublishStreamMessageResponse{}, err
}

func newTestStreamsClient(fake *fakeV1StreamsClient) *streamsClientImpl {
	return &streamsClientImpl{
		client:    fake,
		ctx:       newContextLoader("token", nil),
		seqStates: make(map[string]*producerSeqState),
	}
}

func TestStreamsPublish(t *testing.T) {
	gap := status.Error(codes.FailedPrecondition, "gap")

	cases := map[string]struct {
		errs []error
		// what each Publish call returns, one payload per call
		wantErrs []bool
		check    func(t *testing.T, reqs []*sharedcontracts.PublishStreamMessageRequest)
	}{
		"an ambiguous failure rotates the producer": {
			errs:     []error{nil, status.Error(codes.DeadlineExceeded, "timed out")},
			wantErrs: []bool{false, true, false},
			check: func(t *testing.T, reqs []*sharedcontracts.PublishStreamMessageRequest) {
				require.Len(t, reqs, 3)
				assert.Equal(t, reqs[0].ProducerId, reqs[1].ProducerId)
				assert.Equal(t, int64(1), reqs[1].ProducerSeq)
				assert.NotEqual(t, reqs[1].ProducerId, reqs[2].ProducerId, "a seq that may have landed must not be reused for a different payload")
				assert.Equal(t, int64(0), reqs[2].ProducerSeq)
			},
		},
		"a rejection before storing reuses the seq": {
			errs:     []error{status.Error(codes.ResourceExhausted, "limit")},
			wantErrs: []bool{true, false},
			check: func(t *testing.T, reqs []*sharedcontracts.PublishStreamMessageRequest) {
				require.Len(t, reqs, 2)
				assert.Equal(t, reqs[0].ProducerId, reqs[1].ProducerId)
				assert.Equal(t, int64(0), reqs[1].ProducerSeq)
			},
		},
		"a gap is retried once as a new producer": {
			errs:     []error{nil, gap},
			wantErrs: []bool{false, false},
			check: func(t *testing.T, reqs []*sharedcontracts.PublishStreamMessageRequest) {
				require.Len(t, reqs, 3)
				assert.Equal(t, int64(1), reqs[1].ProducerSeq)
				assert.NotEqual(t, reqs[1].ProducerId, reqs[2].ProducerId)
				assert.Equal(t, int64(0), reqs[2].ProducerSeq)
				assert.Equal(t, reqs[1].Payload, reqs[2].Payload)
			},
		},
		"a repeated gap is returned": {
			errs:     []error{gap, gap},
			wantErrs: []bool{true},
			check: func(t *testing.T, reqs []*sharedcontracts.PublishStreamMessageRequest) {
				assert.Len(t, reqs, 2, "only one retry")
			},
		},
	}

	for label, tc := range cases {
		t.Run(label, func(t *testing.T) {
			fake := &fakeV1StreamsClient{errs: tc.errs}
			s := newTestStreamsClient(fake)

			for i, wantErr := range tc.wantErrs {
				err := s.Publish(context.Background(), "", "t", []byte{byte('a' + i)})
				assert.Equal(t, wantErr, err != nil, "publish %d: %v", i, err)
			}

			tc.check(t, fake.reqs)
		})
	}
}

func (f *fakeV1StreamsClient) GetTopicMetadata(_ context.Context, req *sharedcontracts.GetStreamTopicMetadataRequest, _ ...grpc.CallOption) (*sharedcontracts.StreamTopicMetadata, error) {
	return f.metadata[req.Topic], nil
}

func TestStreamsTopicMetadata(t *testing.T) {
	cursor := "v1:abc"
	publishedAt := time.Date(2026, 10, 6, 12, 0, 0, 0, time.UTC)

	fake := &fakeV1StreamsClient{metadata: map[string]*sharedcontracts.StreamTopicMetadata{
		"busy":    {Namespace: "ns", Topic: "busy", TenantId: "tenant", MessageCount: 3, LatestCursor: &cursor, LastPublishedAt: timestamppb.New(publishedAt)},
		"expired": {Namespace: "ns", Topic: "expired", TenantId: "tenant"},
	}}
	s := newTestStreamsClient(fake)

	busy, err := s.TopicMetadata(context.Background(), "ns", "busy")
	require.NoError(t, err)
	assert.Equal(t, int64(3), busy.MessageCount)
	require.NotNil(t, busy.LatestCursor)
	assert.Equal(t, cursor, *busy.LatestCursor)
	require.NotNil(t, busy.LastPublishedAt)
	assert.True(t, publishedAt.Equal(*busy.LastPublishedAt))

	expired, err := s.TopicMetadata(context.Background(), "ns", "expired")
	require.NoError(t, err)
	assert.Zero(t, expired.MessageCount)
	assert.Nil(t, expired.LatestCursor, "no retained message to resume after")
	assert.Nil(t, expired.LastPublishedAt)
}
