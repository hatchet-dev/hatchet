package features

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

const (
	// the server's limit for a payload sent over gRPC (MaxStreamMessagePayloadBytes)
	maxInlineStreamPayloadBytes = 4*1024*1024 - 5*1024

	// the server's limit for an uploaded payload (MaxStreamUploadedPayloadBytes)
	maxUploadedStreamPayloadBytes = 64 * 1024 * 1024

	// the most publishes one PublishBatch request carries; the engine sets no limit of its own
	maxPublishBatchEntries = 500

	// keeps a PublishBatch request under gRPC's 4 MiB limit with the headroom of a single max-size publish
	maxPublishBatchBytes = 4*1024*1024 - 5*1024

	// an entry's tag and length prefixes, rounded up
	publishEntryOverheadBytes = 16

	// an upload's ref, which a batch carries in place of its payload, rounded up
	payloadRefBytes = 256
)

type streamsConfig struct {
	payloadRequestAttempts int
	resubscribeMinDelay    time.Duration
	resubscribeMaxDelay    time.Duration
}

func defaultStreamsConfig() streamsConfig {
	return streamsConfig{
		payloadRequestAttempts: 3,
		resubscribeMinDelay:    250 * time.Millisecond,
		resubscribeMaxDelay:    10 * time.Second,
	}
}

// StreamEvent is a message read from a durable stream topic.
type StreamEvent struct {
	// Payload is the message payload.
	Payload []byte
	// Cursor resumes Subscribe right after this event.
	Cursor string
	// CreatedAt is when the message was stored.
	CreatedAt time.Time
}

// StreamTopicMetadata describes a topic's retained messages; counts and the latest
// message cover the tenant's retention.
type StreamTopicMetadata struct {
	Namespace    string
	Topic        string
	TenantId     string
	MessageCount int64
	// LatestCursor resumes Subscribe after the newest retained message; nil when none is retained.
	LatestCursor *string
	// LastPublishedAt is when the newest retained message was stored; nil when none is retained.
	LastPublishedAt *time.Time
}

type streamCallOpts struct {
	namespace    string
	cursor       *string
	batchTimeout time.Duration
}

// StreamCallOpt configures a call on the StreamsClient.
type StreamCallOpt func(*streamCallOpts)

// WithStreamNamespace sets the topic's namespace; the empty namespace when omitted.
func WithStreamNamespace(namespace string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.namespace = namespace
	}
}

// WithStreamCursor makes Subscribe resume right after the message the cursor came from, or
// Reset start new readers after it. Ignored by Publish and TopicMetadata.
func WithStreamCursor(cursor string) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.cursor = &cursor
	}
}

// WithStreamBatchTimeout lets Publish hold the message up to d for later publishes to the
// same topic to share its request, bounding the latency batching adds. 0, the default, sends
// it as soon as no other request to the topic is in flight.
func WithStreamBatchTimeout(d time.Duration) StreamCallOpt {
	return func(o *streamCallOpts) {
		o.batchTimeout = d
	}
}

func applyStreamOpts(opts []StreamCallOpt) *streamCallOpts {
	o := &streamCallOpts{}

	for _, opt := range opts {
		opt(o)
	}

	return o
}

// StreamPayloadError is a failed request for a payload too large for a gRPC message.
type StreamPayloadError struct {
	// StatusCode is 0 when the request got no response.
	StatusCode int
	Err        error
}

func (e *StreamPayloadError) Error() string {
	if e.StatusCode == 0 {
		return fmt.Sprintf("stream payload request failed: %v", e.Err)
	}

	return fmt.Sprintf("stream payload request failed with status %d: %v", e.StatusCode, e.Err)
}

func (e *StreamPayloadError) Unwrap() error {
	return e.Err
}

// a request with no response, a 5xx or a 429 may succeed later
func (e *StreamPayloadError) temporary() bool {
	return e.StatusCode == 0 || e.StatusCode >= 500 || e.StatusCode == http.StatusTooManyRequests
}

// the server stored nothing, so the seq is unused
func publishRejectedBeforeStoring(err error) bool {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.ResourceExhausted, codes.Unauthenticated, codes.PermissionDenied,
		// an engine without the RPC
		codes.Unimplemented:
		return true
	default:
		return false
	}
}

func shouldResubscribe(err error, attempt int) bool {
	var payloadErr *StreamPayloadError
	if errors.As(err, &payloadErr) {
		return payloadErr.temporary()
	}

	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.InvalidArgument, codes.PermissionDenied, codes.Unauthenticated, codes.OutOfRange,
		codes.FailedPrecondition, codes.NotFound:
		return false
	case codes.Unimplemented:
		// proxies answer 404 while an engine restarts, but on a first attempt it means no streams support
		return attempt > 0
	default:
		return true
	}
}

type streamProducer struct {
	id  string
	seq int64
}

type uploadResult struct {
	ref string
	err error
}

type queuedPublish struct {
	ctx     context.Context
	payload []byte
	// set instead of an inline payload for one too large for a gRPC message
	uploaded <-chan uploadResult
	sendBy   time.Time
	done     chan error
}

func (p *queuedPublish) requestBytes() int {
	if p.uploaded != nil {
		return payloadRefBytes
	}

	return len(p.payload)
}

// one topic's publishes waiting to be sent
type publishQueue struct {
	items []*queuedPublish
	wake  chan struct{}
}

// StreamsClient publishes to and reads from durable topics. Unlike Runs().SubscribeToStream,
// topics are stored, independent of any run, and readers can resume from a cursor.
type StreamsClient struct {
	grpc     v1.V1StreamsClient
	api      *rest.ClientWithResponses
	tenantId uuid.UUID
	config   streamsConfig

	mu sync.Mutex
	// per (namespace, topic); seq only advances after a publish succeeds
	producers map[string]*streamProducer
	// per (namespace, topic); one request per topic is in flight at a time so two never share a seq
	queues map[string]*publishQueue

	// set once the engine answers PublishBatch as unimplemented: it predates batching
	publishBatchUnsupported atomic.Bool
}

// NewStreamsClient creates a new StreamsClient.
func NewStreamsClient(grpcClient v1.V1StreamsClient, api *rest.ClientWithResponses, tenantId string) *StreamsClient {
	return &StreamsClient{
		grpc:      grpcClient,
		api:       api,
		tenantId:  uuid.MustParse(tenantId),
		config:    defaultStreamsConfig(),
		producers: make(map[string]*streamProducer),
		queues:    make(map[string]*publishQueue),
	}
}

// Publish returns once the message is stored. A client's messages to a topic are delivered in
// the order Publish was called. Publishes to a topic share requests: everything queued while
// one is in flight goes out in the next, and WithStreamBatchTimeout holds a message up to that
// long for more to join it. If ctx ends after the message was sent, Publish returns ctx's error
// though the message may still be stored.
func (s *StreamsClient) Publish(ctx context.Context, topic string, message []byte, opts ...StreamCallOpt) error {
	o := applyStreamOpts(opts)

	// the engine would reject the whole request it shares, failing other publishes too
	if len(message) == 0 {
		return errors.New("payload is empty; a message must carry at least one byte")
	}

	if len(message) > maxUploadedStreamPayloadBytes {
		return fmt.Errorf("payload exceeds maximum size of %d bytes", maxUploadedStreamPayloadBytes)
	}

	if o.batchTimeout < 0 {
		return fmt.Errorf("batch timeout must be 0 or more; got %s", o.batchTimeout)
	}

	p := &queuedPublish{
		ctx:     ctx,
		payload: message,
		sendBy:  time.Now().Add(o.batchTimeout),
		done:    make(chan error, 1),
	}

	// uploaded before taking its place in the topic's order, so uploads overlap
	if len(message) > maxInlineStreamPayloadBytes {
		uploaded := make(chan uploadResult, 1)
		p.uploaded = uploaded

		go func() {
			ref, err := s.uploadPayload(ctx, message)
			uploaded <- uploadResult{ref: ref, err: err}
		}()
	}

	key := o.namespace + "\x00" + topic

	s.mu.Lock()
	q, draining := s.queues[key]
	if !draining {
		q = &publishQueue{wake: make(chan struct{}, 1)}
		s.queues[key] = q
	}
	q.items = append(q.items, p)
	s.mu.Unlock()

	// a new publish may fill the request or have a shorter timeout
	select {
	case q.wake <- struct{}{}:
	default:
	}

	if !draining {
		go s.drainPublishes(key, o.namespace, topic, q)
	}

	select {
	case err := <-p.done:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *StreamsClient) drainPublishes(key, namespace, topic string, q *publishQueue) {
	for {
		batch := s.nextBatch(key, q)
		if batch == nil {
			return
		}

		sent := make([]*queuedPublish, 0, len(batch))
		entries := make([]*v1.StreamPublishEntry, 0, len(batch))

		for _, p := range batch {
			// its caller has given up and nothing was sent, so it never lands
			if err := p.ctx.Err(); err != nil {
				p.done <- err
				continue
			}

			entry := &v1.StreamPublishEntry{Payload: p.payload}

			if p.uploaded != nil {
				// a failed upload fails only its own publish, as the rest never depended on it
				res := <-p.uploaded
				if res.err != nil {
					p.done <- res.err
					continue
				}

				entry = &v1.StreamPublishEntry{PayloadRef: res.ref}
			}

			sent = append(sent, p)
			entries = append(entries, entry)
		}

		if len(entries) > 0 {
			s.sendPublishes(key, namespace, topic, sent, entries)
		}
	}
}

// nextBatch returns the next request's publishes, or nil once the queue is empty and removed.
func (s *StreamsClient) nextBatch(key string, q *publishQueue) []*queuedPublish {
	for {
		s.mu.Lock()

		if len(q.items) == 0 {
			// removed under the lock Publish takes, so a later publish starts a new drain
			delete(s.queues, key)
			s.mu.Unlock()
			return nil
		}

		count := fittingPublishes(q.items)
		full := count < len(q.items) || count == maxPublishBatchEntries

		sendBy := q.items[0].sendBy
		for _, p := range q.items[1:] {
			if p.sendBy.Before(sendBy) {
				sendBy = p.sendBy
			}
		}

		wait := time.Until(sendBy)

		if full || wait <= 0 {
			batch := q.items[:count:count]
			q.items = q.items[count:]
			s.mu.Unlock()
			return batch
		}

		s.mu.Unlock()

		timer := time.NewTimer(wait)
		select {
		case <-q.wake:
		case <-timer.C:
		}
		timer.Stop()
	}
}

// always at least one, so a max-size payload still goes out on its own
func fittingPublishes(items []*queuedPublish) int {
	bytes := 0
	count := 0

	for count < len(items) && count < maxPublishBatchEntries {
		next := items[count].requestBytes() + publishEntryOverheadBytes
		if count > 0 && bytes+next > maxPublishBatchBytes {
			break
		}

		bytes += next
		count++
	}

	return count
}

// sendPublishes sends entries as one request, or one request per entry to an engine that
// predates PublishBatch, and settles each publish.
func (s *StreamsClient) sendPublishes(key, namespace, topic string, sent []*queuedPublish, entries []*v1.StreamPublishEntry) {
	if !s.publishBatchUnsupported.Load() {
		err := s.publishOrdered(key, namespace, topic, entries, true)
		if status.Code(err) != codes.Unimplemented {
			for _, p := range sent {
				p.done <- err
			}

			return
		}

		// nothing was stored, so the batch goes out again below
		s.publishBatchUnsupported.Store(true)
	}

	// settled one by one: a failure partway leaves the publishes before it stored
	for i, p := range sent {
		p.done <- s.publishOrdered(key, namespace, topic, entries[i:i+1], true)
	}
}

func (s *StreamsClient) producer(key string) *streamProducer {
	s.mu.Lock()
	defer s.mu.Unlock()

	producer, ok := s.producers[key]
	if !ok {
		producer = &streamProducer{id: uuid.NewString()}
		s.producers[key] = producer
	}

	return producer
}

func (s *StreamsClient) publishOrdered(key, namespace, topic string, entries []*v1.StreamPublishEntry, retryGap bool) error {
	producer := s.producer(key)

	// shared by every publish in the batch, so no one caller's ctx may cancel it
	ctx := context.Background()

	var err error

	if s.publishBatchUnsupported.Load() && len(entries) == 1 {
		_, err = s.grpc.Publish(ctx, &v1.PublishStreamMessageRequest{
			Namespace:   namespace,
			Topic:       topic,
			Payload:     entries[0].Payload,
			PayloadRef:  entries[0].PayloadRef,
			ProducerId:  producer.id,
			ProducerSeq: producer.seq,
		})
	} else {
		_, err = s.grpc.PublishBatch(ctx, &v1.PublishStreamMessagesRequest{
			Namespace:        namespace,
			Topic:            topic,
			ProducerId:       producer.id,
			FirstProducerSeq: producer.seq,
			Entries:          entries,
		})
	}

	if err != nil {
		// it may still land, and a different payload under its seq would be dropped as a duplicate
		if !publishRejectedBeforeStoring(err) {
			s.mu.Lock()
			s.producers[key] = &streamProducer{id: uuid.NewString()}
			s.mu.Unlock()
		}

		// a gap stored nothing (e.g. the watermark passed cursor retention), so resend as the new producer
		if retryGap && status.Code(err) == codes.FailedPrecondition {
			return s.publishOrdered(key, namespace, topic, entries, false)
		}

		return err
	}

	producer.seq += int64(len(entries))

	return nil
}

func (s *StreamsClient) uploadPayload(ctx context.Context, payload []byte) (string, error) {
	var body bytes.Buffer

	w := multipart.NewWriter(&body)

	part, err := w.CreateFormFile("payload", "payload")
	if err != nil {
		return "", err
	}

	if _, err := part.Write(payload); err != nil {
		return "", err
	}

	if err := w.Close(); err != nil {
		return "", err
	}

	// retries a 5xx or network failure; a 4xx won't succeed on retry
	for attempt := 1; ; attempt++ {
		resp, err := s.api.V1StreamPayloadUploadWithBodyWithResponse(ctx, s.tenantId, w.FormDataContentType(), bytes.NewReader(body.Bytes()))

		var uploadErr *StreamPayloadError

		switch {
		case err != nil:
			uploadErr = &StreamPayloadError{Err: err}
		case resp.JSON200 == nil:
			uploadErr = &StreamPayloadError{StatusCode: resp.StatusCode(), Err: errors.New(string(resp.Body))}
		default:
			return resp.JSON200.Ref, nil
		}

		if ctx.Err() != nil || (uploadErr.StatusCode != 0 && uploadErr.StatusCode < 500) || attempt >= s.config.payloadRequestAttempts {
			return "", uploadErr
		}
	}
}

func (s *StreamsClient) fetchPayload(ctx context.Context, payloadRef string) ([]byte, error) {
	resp, err := s.api.V1StreamPayloadGetWithResponse(ctx, s.tenantId, &rest.V1StreamPayloadGetParams{Ref: payloadRef})
	if err != nil {
		return nil, &StreamPayloadError{Err: err}
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, &StreamPayloadError{StatusCode: resp.StatusCode(), Err: errors.New(string(resp.Body))}
	}

	return resp.Body, nil
}

// TopicMetadata describes topic's retained messages. It returns a NotFound gRPC error for a
// topic nothing was published to.
func (s *StreamsClient) TopicMetadata(ctx context.Context, topic string, opts ...StreamCallOpt) (*StreamTopicMetadata, error) {
	o := applyStreamOpts(opts)

	res, err := s.grpc.GetTopicMetadata(ctx, &v1.GetStreamTopicMetadataRequest{Namespace: o.namespace, Topic: topic})
	if err != nil {
		return nil, err
	}

	md := &StreamTopicMetadata{
		Namespace:    res.Namespace,
		Topic:        res.Topic,
		TenantId:     res.TenantId,
		MessageCount: res.MessageCount,
		LatestCursor: res.LatestCursor,
	}

	if res.LastPublishedAt != nil {
		t := res.LastPublishedAt.AsTime()
		md.LastPublishedAt = &t
	}

	return md, nil
}

// Reset makes Subscribe calls without a cursor start after WithStreamCursor's message, or the
// topic's latest message. Readers already iterating, and any call with a cursor, are unaffected.
func (s *StreamsClient) Reset(ctx context.Context, topic string, opts ...StreamCallOpt) error {
	o := applyStreamOpts(opts)

	_, err := s.grpc.Reset(ctx, &v1.ResetStreamTopicRequest{Namespace: o.namespace, Topic: topic, Cursor: o.cursor})

	return err
}

type subscription struct {
	namespace string
	topic     string
	cursor    *string
	failures  int
}

// Subscribe yields topic's messages after WithStreamCursor's message, or from the oldest retained
// (or, once the topic is reset, after the reset). If the connection drops, it resubscribes from
// the last message yielded, so nothing is skipped or repeated. It ends when the server hangs up
// an idle subscription, ctx ends, or the caller stops iterating; it yields an error, then ends,
// on errors a retry can't fix, such as an expired cursor.
func (s *StreamsClient) Subscribe(ctx context.Context, topic string, opts ...StreamCallOpt) iter.Seq2[StreamEvent, error] {
	o := applyStreamOpts(opts)

	return func(yield func(StreamEvent, error) bool) {
		sub := &subscription{namespace: o.namespace, topic: topic, cursor: o.cursor}

		for attempt := 0; ; attempt++ {
			done, err := s.subscribeOnce(ctx, sub, yield)
			if done || ctx.Err() != nil {
				return
			}

			// a nil err means the stream ended without a hangup: the engine went away, e.g. it is shutting down
			if err != nil && !shouldResubscribe(err, attempt) {
				yield(StreamEvent{}, err)
				return
			}

			delay := min(s.config.resubscribeMinDelay<<sub.failures, s.config.resubscribeMaxDelay)
			if sub.failures < 16 {
				sub.failures++
			}

			timer := time.NewTimer(time.Duration(float64(delay) * (0.5 + rand.Float64()/2))) //nolint:gosec // jitter, not security
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
		}
	}
}

// Events is an alias of Subscribe, which replaces it; Events will be removed in a future release.
func (s *StreamsClient) Events(ctx context.Context, topic string, opts ...StreamCallOpt) iter.Seq2[StreamEvent, error] {
	return s.Subscribe(ctx, topic, opts...)
}

// subscribeOnce yields one subscription's events, and reports whether the iteration is over.
func (s *StreamsClient) subscribeOnce(ctx context.Context, sub *subscription, yield func(StreamEvent, error) bool) (bool, error) {
	// cancels the stream when the caller stops iterating
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	stream, err := s.grpc.Subscribe(ctx, &v1.SubscribeStreamRequest{Namespace: sub.namespace, Topic: sub.topic, Cursor: sub.cursor})
	if err != nil {
		return false, err
	}

	for {
		msg, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return false, nil
		}

		if err != nil {
			return false, err
		}

		if msg.Hangup {
			return true, nil
		}

		for _, entry := range msg.Entries {
			payload := entry.Payload

			if entry.PayloadRef != "" {
				payload, err = s.fetchPayload(ctx, entry.PayloadRef)
				if err != nil {
					return false, err
				}
			}

			cursor := entry.Cursor
			sub.cursor = &cursor
			// reset only on delivery, so a payload fetch failing on every reconnect still backs off
			sub.failures = 0

			event := StreamEvent{Payload: payload, Cursor: entry.Cursor}
			if entry.CreatedAt != nil {
				event.CreatedAt = entry.CreatedAt.AsTime()
			}

			if !yield(event, nil) {
				return true, nil
			}
		}
	}
}
