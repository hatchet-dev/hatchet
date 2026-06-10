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

func TestNewStandaloneTask_WithCron(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("cron-task", sampleTaskFn, WithCron("*/5 * * * *"))
	req, _, _, _ := task.Dump()
	assert.Equal(t, []string{"*/5 * * * *"}, req.CronTriggers)
}

func TestNewStandaloneTask_WithEvents(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("event-task", sampleTaskFn, WithEvents("user:created"))
	req, _, _, _ := task.Dump()
	assert.Equal(t, []string{"user:created"}, req.EventTriggers)
}

func TestNewStandaloneDurableTask_WithCron(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneDurableTask("durable-cron", sampleDurableFn, WithCron("0 0 * * *"))
	req, _, _, _ := task.Dump()
	assert.Equal(t, []string{"0 0 * * *"}, req.CronTriggers)
}

func TestNewStandaloneTask_MultipleCrons(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("multi-cron", sampleTaskFn, WithCron("*/5 * * * *", "0 0 * * *"))
	req, _, _, _ := task.Dump()
	assert.Equal(t, []string{"*/5 * * * *", "0 0 * * *"}, req.CronTriggers)
}

func TestNewStandaloneTask_NoTriggers(t *testing.T) {
	c := newTestClient()
	task := c.NewStandaloneTask("plain", sampleTaskFn)
	req, _, _, _ := task.Dump()
	assert.Empty(t, req.CronTriggers)
	assert.Empty(t, req.EventTriggers)
}
