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
