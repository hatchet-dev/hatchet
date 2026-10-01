package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sharedcontracts "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
)

type fakeV1StreamsClient struct {
	sharedcontracts.V1StreamsClient

	errs []error
	reqs []*sharedcontracts.PublishStreamMessageRequest
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

func TestStreamsPublish_AmbiguousFailureRotatesProducer(t *testing.T) {
	fake := &fakeV1StreamsClient{errs: []error{nil, status.Error(codes.DeadlineExceeded, "timed out")}}
	s := newTestStreamsClient(fake)

	require.NoError(t, s.Publish(context.Background(), "", "t", []byte("a")))
	require.Error(t, s.Publish(context.Background(), "", "t", []byte("b")))
	require.NoError(t, s.Publish(context.Background(), "", "t", []byte("c")))

	require.Len(t, fake.reqs, 3)
	assert.Equal(t, fake.reqs[0].ProducerId, fake.reqs[1].ProducerId)
	assert.Equal(t, int64(1), fake.reqs[1].ProducerSeq)
	assert.NotEqual(t, fake.reqs[1].ProducerId, fake.reqs[2].ProducerId, "a seq that may have landed must not be reused for a different payload")
	assert.Equal(t, int64(0), fake.reqs[2].ProducerSeq)
}

func TestStreamsPublish_RejectedBeforeEnqueueReusesSeq(t *testing.T) {
	fake := &fakeV1StreamsClient{errs: []error{status.Error(codes.ResourceExhausted, "limit")}}
	s := newTestStreamsClient(fake)

	require.Error(t, s.Publish(context.Background(), "", "t", []byte("a")))
	require.NoError(t, s.Publish(context.Background(), "", "t", []byte("a")))

	require.Len(t, fake.reqs, 2)
	assert.Equal(t, fake.reqs[0].ProducerId, fake.reqs[1].ProducerId)
	assert.Equal(t, int64(0), fake.reqs[1].ProducerSeq)
}

func TestStreamsPublish_GapRetriesOnceAsANewProducer(t *testing.T) {
	fake := &fakeV1StreamsClient{errs: []error{nil, status.Error(codes.FailedPrecondition, "gap")}}
	s := newTestStreamsClient(fake)

	require.NoError(t, s.Publish(context.Background(), "", "t", []byte("a")))
	require.NoError(t, s.Publish(context.Background(), "", "t", []byte("b")), "a gap must be retried, not returned")

	require.Len(t, fake.reqs, 3)
	assert.Equal(t, int64(1), fake.reqs[1].ProducerSeq)
	assert.NotEqual(t, fake.reqs[1].ProducerId, fake.reqs[2].ProducerId)
	assert.Equal(t, int64(0), fake.reqs[2].ProducerSeq)
	assert.Equal(t, []byte("b"), fake.reqs[2].Payload)
}

func TestStreamsPublish_RepeatedGapIsReturned(t *testing.T) {
	gap := status.Error(codes.FailedPrecondition, "gap")
	fake := &fakeV1StreamsClient{errs: []error{gap, gap}}
	s := newTestStreamsClient(fake)

	assert.Equal(t, codes.FailedPrecondition, status.Code(s.Publish(context.Background(), "", "t", []byte("a"))))
	assert.Len(t, fake.reqs, 2, "only one retry")
}
