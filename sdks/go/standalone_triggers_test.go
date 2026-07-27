//go:build !e2e && !load && !rampup && !integration

package hatchet

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	v0Client "github.com/hatchet-dev/hatchet/pkg/client"
	cloudrest "github.com/hatchet-dev/hatchet/pkg/client/cloud/rest"
	"github.com/hatchet-dev/hatchet/pkg/client/rest"
)

// stubV0Client implements v0Client.Client with minimal stubs so that
// NewStandaloneTask can build a declaration without a live server.
type stubV0Client struct{}

func (s *stubV0Client) Admin() v0Client.AdminClient              { return nil }
func (s *stubV0Client) Cron() v0Client.CronClient                { return nil }
func (s *stubV0Client) Schedule() v0Client.ScheduleClient        { return nil }
func (s *stubV0Client) Dispatcher() v0Client.DispatcherClient    { return nil }
func (s *stubV0Client) Event() v0Client.EventClient              { return nil }
func (s *stubV0Client) Subscribe() v0Client.SubscribeClient      { return nil }
func (s *stubV0Client) API() *rest.ClientWithResponses           { return nil }
func (s *stubV0Client) CloudAPI() *cloudrest.ClientWithResponses { return nil }
func (s *stubV0Client) Logger() *zerolog.Logger                  { l := zerolog.Nop(); return &l }
func (s *stubV0Client) TenantId() string                         { return "00000000-0000-0000-0000-000000000000" }
func (s *stubV0Client) Namespace() string                        { return "" }
func (s *stubV0Client) CloudRegisterID() *string                 { return nil }
func (s *stubV0Client) RunnableActions() []string                { return nil }

// newTestClient returns a Client backed by a stub, suitable for
// offline tests with no server
func newTestClient() *Client {
	return &Client{legacyClient: &stubV0Client{}}
}

// sampleTaskFn is a minimal function matching the standalone task signature.
func sampleTaskFn(_ Context, _ any) (any, error) { return nil, nil }

// sampleDurableFn is a minimal function matching the standalone durable task signature.
func sampleDurableFn(_ DurableContext, _ any) (any, error) { return nil, nil }

// WithCron and WithEvents on standalone tasks should appear in the registration request.

func TestNewStandaloneTask_Triggers(t *testing.T) {
	tests := []struct {
		name       string
		taskName   string
		options    []StandaloneTaskOption
		wantCron   []string
		wantEvents []string
	}{
		{
			name:     "with cron",
			taskName: "cron-task",
			options:  []StandaloneTaskOption{WithCron("*/5 * * * *")},
			wantCron: []string{"*/5 * * * *"},
		},
		{
			name:       "with events",
			taskName:   "event-task",
			options:    []StandaloneTaskOption{WithEvents("user:created")},
			wantEvents: []string{"user:created"},
		},
		{
			name:     "multiple crons",
			taskName: "multi-cron",
			options:  []StandaloneTaskOption{WithCron("*/5 * * * *", "0 0 * * *")},
			wantCron: []string{"*/5 * * * *", "0 0 * * *"},
		},
		{
			name:     "no triggers",
			taskName: "plain",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient()
			task := c.NewStandaloneTask(tt.taskName, sampleTaskFn, tt.options...)
			req, _, _, _ := task.Dump()
			assert.Equal(t, tt.wantCron, req.CronTriggers)
			assert.Equal(t, tt.wantEvents, req.EventTriggers)
		})
	}
}

func TestNewStandaloneDurableTask_WithCron(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneDurableTask("durable-cron", sampleDurableFn, WithCron("0 0 * * *"))
	req, _, _, _ := task.Dump()
	assert.Equal(t, []string{"0 0 * * *"}, req.CronTriggers)
}
