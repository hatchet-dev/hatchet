package workflows

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hatchet-dev/hatchet/api/v1/server/oas/gen"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/pkg/config/database"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type fakePauseRepo struct {
	repository.Repository
}

func (r fakePauseRepo) Workflows() repository.WorkflowRepository { return fakePauseWorkflows{} }

type fakePauseWorkflows struct {
	repository.WorkflowRepository
}

func (fakePauseWorkflows) PauseWorkflow(context.Context, uuid.UUID, uuid.UUID, repository.PauseWorkflowOpts) (*sqlcv1.Workflow, error) {
	return &sqlcv1.Workflow{}, nil
}

func (fakePauseWorkflows) UnpauseWorkflow(context.Context, uuid.UUID, uuid.UUID) (*sqlcv1.Workflow, error) {
	return &sqlcv1.Workflow{}, nil
}

type recordingMQ struct {
	msgqueue.MessageQueue
	sent []string
}

func (m *recordingMQ) SendMessage(_ context.Context, q msgqueue.Queue, msg *msgqueue.Message) error {
	m.sent = append(m.sent, q.Name()+"/"+msg.ID)
	return nil
}

type recordingPubSub struct {
	msgqueue.PubSub
	pubs []string
}

func (p *recordingPubSub) Pub(_ context.Context, topic msgqueue.Topic, msg *msgqueue.Message) error {
	p.pubs = append(p.pubs, topic.Name()+"/"+msg.ID)
	return nil
}

func TestApplyPauseWorkflowRequestRouting(t *testing.T) {
	var pause, unpause gen.PauseWorkflowRequest

	require.NoError(t, pause.FromPauseWorkflowRequestPause(gen.PauseWorkflowRequestPause{
		Action:                                  gen.Pause,
		PausedWorkflowCronRunQueueBehavior:      gen.QUEUE,
		PausedWorkflowScheduledRunQueueBehavior: gen.QUEUE,
		PausedWorkflowQueueTTL:                  "1h",
	}))
	require.NoError(t, unpause.FromPauseWorkflowRequestUnpause(gen.PauseWorkflowRequestUnpause{Action: gen.Unpause}))

	tests := []struct {
		name     string
		request  gen.PauseWorkflowRequest
		wantSent []string
		wantPubs []string
	}{
		{
			name:     "pause is durable",
			request:  pause,
			wantSent: []string{"task_processing_queue_v2/toggle-pause-workflow"},
		},
		{
			name:     "unpause is a best-effort wake",
			request:  unpause,
			wantPubs: []string{"task_controller_wake_v1/toggle-pause-workflow"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := zerolog.Nop()
			mq := &recordingMQ{}
			ps := &recordingPubSub{}

			svc := NewWorkflowService(&server.ServerConfig{
				Layer:          &database.Layer{V1: fakePauseRepo{}},
				Logger:         &l,
				MessageQueueV1: mq,
				PubSubV1:       ps,
			})

			_, err := svc.applyPauseWorkflowRequest(context.Background(), uuid.New(), uuid.New(), tt.request)
			require.NoError(t, err)

			assert.Equal(t, tt.wantSent, mq.sent)
			assert.Equal(t, tt.wantPubs, ps.pubs)
		})
	}
}
