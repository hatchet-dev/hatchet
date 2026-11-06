package postgres

import (
	"context"
	"encoding/json"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
)

type notifyRecordingRepo struct {
	v1.MessageQueueRepository
	mu       sync.Mutex
	notified []msgqueue.Message
}

func (r *notifyRecordingRepo) BindQueue(context.Context, string, bool, bool, bool, *string) error {
	return nil
}

func (r *notifyRecordingRepo) Notify(_ context.Context, _ string, payload string) error {
	var msg msgqueue.Message

	if err := json.Unmarshal([]byte(payload), &msg); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.notified = append(r.notified, msg)

	return nil
}

func TestPubSendsOneNotifyPerPayload(t *testing.T) {
	tests := []struct {
		name     string
		payloads [][]byte
		want     int
	}{
		{name: "no payloads", payloads: nil, want: 1},
		{name: "one payload", payloads: [][]byte{[]byte(`{}`)}, want: 1},
		{name: "three payloads", payloads: [][]byte{[]byte(`{}`), []byte(`{}`), []byte(`{}`)}, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &notifyRecordingRepo{}

			cleanup, ps, err := NewPubSub(repo)
			require.NoError(t, err)
			defer func() { require.NoError(t, cleanup()) }()

			require.NoError(t, ps.Pub(context.Background(), msgqueue.CronRefreshTopic(), &msgqueue.Message{ID: msgqueue.MsgIDCronUpdate, Payloads: tt.payloads}))

			require.Len(t, repo.notified, tt.want)

			for _, msg := range repo.notified {
				assert.Equal(t, msgqueue.MsgIDCronUpdate, msg.ID)
				assert.LessOrEqual(t, len(msg.Payloads), 1)
			}
		})
	}
}
