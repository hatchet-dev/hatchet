package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/client/create"
	"github.com/hatchet-dev/hatchet/pkg/worker/condition"
)

type HatchetWorkerContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	ID() string

	GetLabels() map[string]interface{}

	UpsertLabels(labels map[string]interface{}) error

	HasWorkflow(workflowName string) bool
}

type HatchetContext interface {
	context.Context

	SetContext(ctx context.Context)

	GetContext() context.Context

	Worker() HatchetWorkerContext

	StepOutput(step string, target interface{}) error

	TriggerDataKeys() []string

	TriggerData(key string, target interface{}) error

	StepRunErrors() map[string]string

	TriggeredByEvent() bool

	WorkflowInput(target interface{}) error

	UserData(target interface{}) error

	AdditionalMetadata() map[string]string

	StepName() string

	StepRunId() string

	StepId() string

	WorkflowRunId() string

	WorkflowId() *string

	WorkflowVersionId() *string

	Log(message string)

	StreamEvent(message []byte)

	PutStream(message string)

	SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.Workflow, error)

	SpawnWorkflows(childWorkflows []*SpawnWorkflowsOpts) ([]*client.Workflow, error)

	ReleaseSlot() error

	RefreshTimeout(incrementTimeoutBy string) error

	RetryCount() int

	ParentOutput(parent create.NamedTask, output interface{}) error

	client() client.Client

	action() *client.Action

	CurChildIndex() int
	IncChildIndex()

	Priority() int32

	FilterPayload() map[string]interface{}
}

// Deprecated: TriggeredBy is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type TriggeredBy string

// Deprecated: These constants are part of the legacy v0 workflow definition system.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead. Migration guide: https://docs.hatchet.run/home/migration-guide-go
const (
	TriggeredByEvent    TriggeredBy = "event"
	TriggeredByCron     TriggeredBy = "cron"
	TriggeredBySchedule TriggeredBy = "schedule"
)

// Deprecated: JobRunLookupData is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type JobRunLookupData struct {
	Input       map[string]interface{} `json:"input"`
	TriggeredBy TriggeredBy            `json:"triggered_by"`
	Steps       map[string]StepData    `json:"steps,omitempty"`
}

// Deprecated: StepRunData is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type StepRunData struct {
	Input              map[string]interface{}            `json:"input"`
	TriggeredBy        TriggeredBy                       `json:"triggered_by"`
	Parents            map[string]StepData               `json:"parents"`
	Triggers           map[string]map[string]interface{} `json:"triggers,omitempty"`
	AdditionalMetadata map[string]string                 `json:"additional_metadata"`
	UserData           map[string]interface{}            `json:"user_data"`
	StepRunErrors      map[string]string                 `json:"step_run_errors,omitempty"`
}

// Deprecated: StepData is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type StepData map[string]interface{}

type hatchetContext struct {
	context.Context

	w *hatchetWorkerContext

	a        *client.Action
	stepData *StepRunData
	c        client.Client
	l        *zerolog.Logger

	i          int
	indexMu    sync.Mutex
	listener   *client.WorkflowRunsListener
	listenerMu sync.Mutex

	streamEventIndex   int64
	streamEventIndexMu sync.Mutex
}

type hatchetWorkerContext struct {
	context.Context
	id     *string
	worker *Worker
}

func newHatchetContext(
	ctx context.Context,
	action *client.Action,
	client client.Client,
	l *zerolog.Logger,
	w *Worker,
) (HatchetContext, error) {
	c := &hatchetContext{
		Context: ctx,
		a:       action,
		c:       client,
		l:       l,
		w: &hatchetWorkerContext{
			Context: ctx,
			id:      w.id,
			worker:  w,
		},
	}

	if action.GetGroupKeyRunId != "" {
		err := c.populateStepDataForGroupKeyRun()

		if err != nil {
			return nil, err
		}
	} else {
		err := c.populateStepData()

		if err != nil {
			return nil, err
		}
	}

	return c, nil
}

func (h *hatchetContext) client() client.Client {
	return h.c
}

func (h *hatchetContext) action() *client.Action {
	return h.a
}

func (h *hatchetContext) Worker() HatchetWorkerContext {
	return h.w
}

func (h *hatchetContext) SetContext(ctx context.Context) {
	h.Context = ctx
}

func (h *hatchetContext) GetContext() context.Context {
	return h.Context
}

func (h *hatchetContext) StepOutput(step string, target interface{}) error {
	if val, ok := h.stepData.Parents[step]; ok {
		return toTarget(val, target)
	}

	return fmt.Errorf("step %s not found in action payload", step)
}

func (h *hatchetContext) TriggerDataKeys() []string {
	keys := make([]string, 0, len(h.stepData.Triggers))

	for k := range h.stepData.Triggers {
		keys = append(keys, k)
	}

	return keys
}

func (h *hatchetContext) TriggerData(key string, target interface{}) error {
	if val, ok := h.stepData.Triggers[key]; ok {
		return toTarget(val, target)
	}

	return fmt.Errorf("trigger %s not found in action payload", key)
}

func (h *hatchetContext) ParentOutput(parent create.NamedTask, output interface{}) error {
	stepName := parent.GetName()

	if val, ok := h.stepData.Parents[stepName]; ok {
		return toTarget(val, output)
	}

	return fmt.Errorf("parent %s not found in action payload", stepName)
}

// Deprecated: TriggeredByEvent is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) TriggeredByEvent() bool {
	return h.stepData.TriggeredBy == TriggeredByEvent
}

// Deprecated: WorkflowInput is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) WorkflowInput(target interface{}) error {
	return toTarget(h.stepData.Input, target)
}

// Deprecated: StepRunErrors is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) StepRunErrors() map[string]string {
	errors := h.stepData.StepRunErrors

	if len(errors) == 0 {
		h.l.Error().Msg("No step run errors found. `ctx.StepRunErrors` is intended to be run in an on-failure step, and will only work on engine versions more recent than v0.53.10")
	}

	return errors
}

// Deprecated: UserData is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) UserData(target interface{}) error {
	return toTarget(h.stepData.UserData, target)
}

// Deprecated: FilterPayload is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) FilterPayload() map[string]interface{} {
	payload := h.stepData.Triggers["filter_payload"]

	return payload
}

// Deprecated: AdditionalMetadata is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) AdditionalMetadata() map[string]string {
	return h.stepData.AdditionalMetadata
}

// Deprecated: StepName is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) StepName() string {
	return h.a.StepName
}

// Deprecated: StepRunId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) StepRunId() string {
	return h.a.StepRunId
}

// Deprecated: StepId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) StepId() string {
	return h.a.StepId
}

// Deprecated: WorkflowRunId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) WorkflowRunId() string {
	return h.a.WorkflowRunId
}

// Deprecated: WorkflowId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) WorkflowId() *string {
	return h.a.WorkflowId
}

// Deprecated: WorkflowVersionId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) WorkflowVersionId() *string {
	return h.a.WorkflowVersionId
}

// Deprecated: Log is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) Log(message string) {
	infoLevel := "INFO"

	runes := []rune(message)

	if len(runes) > 10_000 {
		h.l.Warn().Msg("log message is too long, truncating to the first 10,000 characters")
		message = string(runes[:10_000])
	}

	stepRunId := h.a.StepRunId
	retryCount := h.a.RetryCount
	createdAt := timestamppb.Now()

	go func() {
		const maxRetries = 3
		baseDelay := 100 * time.Millisecond

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var err error

		for attempt := range maxRetries + 1 {
			if attempt > 0 {
				delay := baseDelay * time.Duration(1<<(attempt-1))

				select {
				case <-ctx.Done():
					h.l.Warn().Err(err).Msg("log delivery timed out, abandoning")
					return
				case <-time.After(delay):
				}
			}

			err = h.c.Event().PutLogWithTimestamp(ctx, stepRunId, message, &infoLevel, &retryCount, createdAt)
			if err == nil {
				return
			}

			h.l.Warn().Err(err).Msgf("failed to put log (attempt %d/%d)", attempt+1, maxRetries+1)
		}

		h.l.Err(err).Msg("could not put log after all retries")
	}()
}

// Deprecated: ReleaseSlot is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) ReleaseSlot() error {
	err := h.c.Dispatcher().ReleaseSlot(h, h.a.StepRunId)

	if err != nil {
		return fmt.Errorf("failed to release slot: %w", err)
	}

	return nil
}

// Deprecated: RefreshTimeout is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) RefreshTimeout(incrementTimeoutBy string) error {
	err := h.c.Dispatcher().RefreshTimeout(h, h.a.StepRunId, incrementTimeoutBy)

	if err != nil {
		return fmt.Errorf("failed to refresh timeout: %w", err)
	}

	return nil
}

// Deprecated: StreamEvent is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) StreamEvent(message []byte) {
	h.streamEventIndexMu.Lock()
	currentIndex := h.streamEventIndex
	h.streamEventIndex++
	h.streamEventIndexMu.Unlock()

	err := h.c.Event().PutStreamEvent(h, h.a.StepRunId, message, client.WithStreamEventIndex(currentIndex))

	if err != nil {
		h.l.Err(err).Msg("could not put stream event")
	}
}

// Deprecated: PutStream is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) PutStream(message string) {
	h.StreamEvent([]byte(message))
}

// Deprecated: RetryCount is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) RetryCount() int {
	return int(h.a.RetryCount)
}

// Deprecated: CurChildIndex is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) CurChildIndex() int {
	h.indexMu.Lock()
	defer h.indexMu.Unlock()

	return h.i
}

// Deprecated: IncChildIndex is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) IncChildIndex() {
	h.indexMu.Lock()
	h.i++
	h.indexMu.Unlock()
}

// Deprecated: SpawnWorkflowOpts is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type SpawnWorkflowOpts struct {
	Key                *string
	Sticky             *bool
	AdditionalMetadata *map[string]string
	Priority           *int32
}

func (h *hatchetContext) saveOrLoadListener() (*client.WorkflowRunsListener, error) {
	return h.client().Subscribe().SubscribeToWorkflowRunEvents(h)
}

// Deprecated: SpawnWorkflow is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) SpawnWorkflow(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.Workflow, error) {
	if opts == nil {
		opts = &SpawnWorkflowOpts{}
	}

	var desiredWorker *string

	if opts.Sticky != nil {
		if _, exists := h.w.worker.registered_workflows[workflowName]; !exists {
			return nil, fmt.Errorf("cannot run with sticky: workflow %s is not registered on this worker", workflowName)
		}

		desiredWorker = h.w.id
	}

	listener, err := h.saveOrLoadListener()

	if err != nil {
		return nil, err
	}

	if ns := h.client().Namespace(); ns != "" && !strings.HasPrefix(workflowName, ns) {
		workflowName = fmt.Sprintf("%s%s", ns, workflowName)
	}

	h.indexMu.Lock()
	childIndex := h.i
	h.i++
	h.indexMu.Unlock()

	workflowRunId, err := h.client().Admin().RunChildWorkflow(
		workflowName,
		input,
		&client.ChildWorkflowOpts{
			ParentId:           h.WorkflowRunId(),
			ParentTaskRunId:    h.StepRunId(),
			ChildIndex:         childIndex,
			ChildKey:           opts.Key,
			DesiredWorkerId:    desiredWorker,
			AdditionalMetadata: opts.AdditionalMetadata,
			Priority:           opts.Priority,
		},
	)

	if err != nil {
		return nil, fmt.Errorf("failed to spawn workflow: %w", err)
	}

	return client.NewWorkflow(workflowRunId, listener), nil
}

// Deprecated: SpawnWorkflowsOpts is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type SpawnWorkflowsOpts struct {
	WorkflowName       string
	Input              any
	Key                *string
	Sticky             *bool
	AdditionalMetadata *map[string]string
}

// Deprecated: SpawnWorkflows is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) SpawnWorkflows(childWorkflows []*SpawnWorkflowsOpts) ([]*client.Workflow, error) {

	triggerWorkflows := make([]*client.RunChildWorkflowsOpts, len(childWorkflows))
	listener, err := h.saveOrLoadListener()

	for i, c := range childWorkflows {

		var desiredWorker *string

		if c.Sticky != nil {
			if _, exists := h.w.worker.registered_workflows[c.WorkflowName]; !exists {
				return nil, fmt.Errorf("cannot run with sticky: workflow %s is not registered on this worker", c.WorkflowName)
			}

			desiredWorker = h.w.id
		}

		if err != nil {
			return nil, err
		}
		workflowName := c.WorkflowName

		if ns := h.client().Namespace(); ns != "" && !strings.HasPrefix(c.WorkflowName, ns) {
			workflowName = fmt.Sprintf("%s%s", ns, workflowName)
		}

		h.indexMu.Lock()
		childIndex := h.i
		h.i++
		h.indexMu.Unlock()

		triggerWorkflows[i] = &client.RunChildWorkflowsOpts{
			WorkflowName: workflowName,
			Input:        c.Input,
			Opts: &client.ChildWorkflowOpts{
				ParentId:           h.WorkflowRunId(),
				ParentTaskRunId:    h.StepRunId(),
				ChildIndex:         childIndex,
				ChildKey:           c.Key,
				DesiredWorkerId:    desiredWorker,
				AdditionalMetadata: c.AdditionalMetadata,
			},
		}
	}

	workflowRunIds, err := h.client().Admin().RunChildWorkflows(
		triggerWorkflows,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to spawn workflow: %w", err)
	}

	createdWorkflows := make([]*client.Workflow, len(workflowRunIds))

	for i, workflowRunId := range workflowRunIds {
		createdWorkflows[i] = client.NewWorkflow(workflowRunId, listener)
	}

	return createdWorkflows, nil
}

// Deprecated: ChildIndex is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) ChildIndex() *int32 {
	return h.a.ChildIndex
}

// Deprecated: ChildKey is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) ChildKey() *string {
	return h.a.ChildKey
}

// Deprecated: ParentWorkflowRunId is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) ParentWorkflowRunId() *string {
	return h.a.ParentWorkflowRunId
}

// Deprecated: Priority is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) Priority() int32 {
	return h.a.Priority
}

func (h *hatchetContext) populateStepDataForGroupKeyRun() error {
	if h.stepData != nil {
		return nil
	}

	inputData := map[string]interface{}{}

	err := json.Unmarshal(h.a.ActionPayload, &inputData)

	if err != nil {
		return err
	}

	h.stepData = &StepRunData{
		Input: inputData,
	}

	return nil
}

func (h *hatchetContext) populateStepData() error {
	if h.stepData != nil {
		return nil
	}

	h.stepData = &StepRunData{}

	jsonBytes := h.a.ActionPayload

	if len(jsonBytes) == 0 {
		jsonBytes = []byte("{}")
	}

	err := json.Unmarshal(jsonBytes, h.stepData)

	if err != nil {
		return err
	}

	h.stepData.AdditionalMetadata = h.a.AdditionalMetadata

	return nil
}

func toTarget(data interface{}, target interface{}) error {
	dataBytes, err := json.Marshal(data)

	if err != nil {
		return err
	}

	err = json.Unmarshal(dataBytes, target)

	if err != nil {
		return err
	}

	return nil
}

// Deprecated: SetContext is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) SetContext(ctx context.Context) {
	wc.Context = ctx
}

// Deprecated: GetContext is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) GetContext() context.Context {
	return wc.Context
}

// Deprecated: ID is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) ID() string {
	if wc.id == nil {
		return ""
	}

	return *wc.id
}

// Deprecated: GetLabels is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) GetLabels() map[string]interface{} {
	return wc.worker.labels
}

// Deprecated: UpsertLabels is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) UpsertLabels(labels map[string]interface{}) error {

	if wc.id == nil {
		return fmt.Errorf("worker id is nil, cannot upsert labels (are on web worker?)")
	}

	err := wc.worker.client.Dispatcher().UpsertWorkerLabels(wc.Context, *wc.id, labels)

	if err != nil {
		return fmt.Errorf("failed to upsert labels: %w", err)
	}

	wc.worker.labels = labels
	return nil
}

// Deprecated: HasWorkflow is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (wc *hatchetWorkerContext) HasWorkflow(workflowName string) bool {
	return wc.worker.registered_workflows[workflowName]
}

// Deprecated: SingleWaitResult is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type SingleWaitResult struct {
	*WaitResult

	key string
}

func newSingleWaitResult(key string, wr *WaitResult) *SingleWaitResult {
	return &SingleWaitResult{
		WaitResult: wr,
		key:        key,
	}
}

// Deprecated: Unmarshal is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (w *SingleWaitResult) Unmarshal(in interface{}) error {
	return w.WaitResult.Unmarshal(w.key, in)
}

// Deprecated: WaitResult is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type WaitResult struct {
	allResults map[string]map[string][]map[string]interface{}
}

func newWaitResult(dataBytes []byte) (*WaitResult, error) {
	var allResults map[string]map[string][]map[string]interface{}

	err := json.Unmarshal(dataBytes, &allResults)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal wait result: %w", err)
	}

	return &WaitResult{
		allResults: allResults,
	}, nil
}

// Deprecated: ErrMarshalKeyNotFound is an internal type used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
type ErrMarshalKeyNotFound struct {
	Key string
}

// Deprecated: Error is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (e ErrMarshalKeyNotFound) Error() string {
	return fmt.Sprintf("key %s not found", e.Key)
}

// Deprecated: Keys is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (w *WaitResult) Keys() []string {
	keys := make([]string, 0, len(w.allResults))

	for _, v := range w.allResults {
		for k2 := range v {
			keys = append(keys, k2)
		}
	}

	return keys
}

// Deprecated: Unmarshal is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (w *WaitResult) Unmarshal(key string, in interface{}) error {
	eNotFound := ErrMarshalKeyNotFound{
		Key: key,
	}

	if w.allResults == nil {
		return eNotFound
	}

	for _, v := range w.allResults {
		if _, exists := v[key]; exists && len(v[key]) > 0 {
			data, err := json.Marshal(v[key][0])

			if err != nil {
				return fmt.Errorf("failed to marshal data: %w", err)
			}

			err = json.Unmarshal(data, in)

			if err != nil {
				return fmt.Errorf("failed to unmarshal data: %w", err)
			}

			return nil
		}
	}

	return nil
}

// DurableHatchetContext extends HatchetContext with methods for durable tasks.
type DurableHatchetContext interface {
	HatchetContext

	// SleepFor pauses execution for the specified duration and returns after that time has elapsed.
	// Duration is "global" meaning it will wait in real time regardless of transient failures
	// like worker restarts.
	// Example: "10s" for 10 seconds, "1m" for 1 minute, etc.
	SleepFor(duration time.Duration) (*SingleWaitResult, error)

	// TODO: docs
	WaitForEvent(eventKey, expression string) (*SingleWaitResult, error)

	// WaitFor pauses execution until the specified conditions are met.
	// Conditions are "global" meaning they will wait in real time regardless of transient failures
	// like worker restarts.
	WaitFor(conditions condition.Condition) (*WaitResult, error)
}

// durableHatchetContext implements the DurableHatchetContext interface.
type durableHatchetContext struct {
	*hatchetContext

	waitKeyCounterMu sync.Mutex
	waitKeyCounter   int

	durableEventListener *client.DurableEventsListener
	durableListenerMu    sync.Mutex
}

// SleepFor implements the DurableHatchetContext.SleepFor method.
func (d *durableHatchetContext) SleepFor(duration time.Duration) (*SingleWaitResult, error) {
	// Implement SleepFor functionality
	// Call appropriate client methods to register a durable event
	c := condition.SleepCondition(duration)

	wr, err := d.WaitFor(c)

	if err != nil {
		return nil, err
	}

	return newSingleWaitResult(c.Key(), wr), nil
}

// WaitForEvent implements the DurableHatchetContext.WaitForEvent method.
func (d *durableHatchetContext) WaitForEvent(eventKey, expression string) (*SingleWaitResult, error) {
	// Implement WaitForEvent functionality
	// Call appropriate client methods to register a durable event
	wr, err := d.WaitFor(condition.UserEventCondition(eventKey, expression))

	if err != nil {
		return nil, err
	}

	return newSingleWaitResult(eventKey, wr), nil
}

// WaitFor implements the DurableHatchetContext.WaitFor method.
func (d *durableHatchetContext) WaitFor(conditions condition.Condition) (*WaitResult, error) {
	// Increment wait key to ensure unique keys for multiple wait operations
	d.waitKeyCounterMu.Lock()
	d.waitKeyCounter++
	count := d.waitKeyCounter
	d.waitKeyCounterMu.Unlock()

	// TODO: MOVE SAVE OR LOAD DURABLE EVENT LISTENER TO THE CLIENT
	durableListener, err := d.saveOrLoadDurableEventListener()

	if err != nil {
		return nil, err
	}

	// compose the durable event to listen for
	c := conditions.ToPB(v1.Action_CREATE)
	signalKey := fmt.Sprintf("signal-%d", count)

	_, err = d.client().Dispatcher().RegisterDurableEvent(d, &v1.RegisterDurableEventRequest{
		TaskId:    d.StepRunId(),
		SignalKey: signalKey,
		Conditions: &v1.DurableEventListenerConditions{
			SleepConditions:     c.SleepConditions,
			UserEventConditions: c.UserEventConditions,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to register durable event: %w", err)
	}

	resCh := make(chan []byte)

	err = durableListener.AddSignal(d.StepRunId(), signalKey, func(e client.DurableEvent) error {
		resCh <- e.Data

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to add signal: %w", err)
	}

	data := <-resCh

	return newWaitResult(data)
}

func (h *durableHatchetContext) saveOrLoadDurableEventListener() (*client.DurableEventsListener, error) {
	return h.client().Subscribe().ListenForDurableEvents(context.Background())
}

// Deprecated: NewDurableHatchetContext is an internal function used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of calling this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func NewDurableHatchetContext(ctx HatchetContext) DurableHatchetContext {
	// Try to cast directly if it's already a DurableHatchetContext
	if durableCtx, ok := ctx.(DurableHatchetContext); ok {
		return durableCtx
	}

	// If it's a hatchetContext, wrap it in a durableHatchetContext
	if hCtx, ok := ctx.(*hatchetContext); ok {
		return &durableHatchetContext{
			hatchetContext: hCtx,
			waitKeyCounter: 0,
		}
	}

	// Create a new wrapper if it's some other implementation
	return &durableHatchetContext{
		hatchetContext: &hatchetContext{
			Context: ctx,
			a:       ctx.action(),
			c:       ctx.client(),
			w:       ctx.Worker().(*hatchetWorkerContext),
		},
		waitKeyCounter: 0,
	}
}

// Deprecated: RunChild is an internal method used by the new Go SDK.
// Use the new Go SDK at github.com/hatchet-dev/hatchet/sdks/go instead of using this directly. Migration guide: https://docs.hatchet.run/home/migration-guide-go
func (h *hatchetContext) RunChild(workflowName string, input any, opts *SpawnWorkflowOpts) (*client.WorkflowResult, error) {
	// Spawn the child workflow
	workflow, err := h.SpawnWorkflow(workflowName, input, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to spawn child workflow: %w", err)
	}

	// Wait for the result
	result, err := workflow.Result()
	if err != nil {
		return nil, fmt.Errorf("child workflow execution failed: %w", err)
	}

	return result, nil
}
