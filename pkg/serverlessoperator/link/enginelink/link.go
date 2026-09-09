// Package enginelink is the in-engine Link: a registration is an engine Worker row created and
// activated in process, an operator-backed dispatcher session that forwards assigned actions,
// and direct calls into the dispatcher for step events and durable invocations. It replays
// the steps grpcoperator.Listen performs server-side without a stream, so the serverless
// operator core runs inside the engine's dispatcher process with the same behaviour it has
// out of process over grpclink.
package enginelink

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	adminv1 "github.com/hatchet-dev/hatchet/internal/services/admin/v1"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	v1 "github.com/hatchet-dev/hatchet/internal/services/shared/proto/v1"
	"github.com/hatchet-dev/hatchet/pkg/client/types"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/serverlessoperator/link"
	"github.com/hatchet-dev/hatchet/pkg/validator"
)

// Dispatcher is the seam to the engine dispatcher, satisfied by *dispatcher.DispatcherImpl.
// It is an interface so the link can be tested without a running dispatcher.
type Dispatcher interface {
	// AddOperatorSession registers op as the operator-backed session for workerId; release
	// removes it.
	AddOperatorSession(workerId uuid.UUID, sessionId uuid.UUID, op operator.Operator) (release func())

	// NotifyNewWorker tells the tenant's scheduler that the worker (or its action set) changed.
	NotifyNewWorker(ctx context.Context, tenant *sqlcv1.Tenant, workerId uuid.UUID)

	// SendStepActionEvent reports task progress; ctx must carry the tenant.
	SendStepActionEvent(ctx context.Context, req *contracts.StepActionEvent) (*contracts.ActionEventResponse, error)

	// RegisterDurableTask opens the in-process durable task session; ctx must carry the tenant.
	RegisterDurableTask(ctx context.Context, externalId uuid.UUID) (chan<- *v1.DurableTaskRequest, <-chan *v1.DurableTaskResponse, error)
}

// tenantStore, operatorStore and workerStore are the repository subsets the link uses, narrow
// so tests substitute doubles without stubbing the whole repository tree.
type tenantStore interface {
	GetTenantByID(ctx context.Context, tenantId uuid.UUID) (*sqlcv1.Tenant, error)
}

type operatorStore interface {
	UpsertServerlessOperator(ctx context.Context, tenantId uuid.UUID, name string) (*sqlcv1.V1Operator, error)
}

type workerStore interface {
	CreateNewWorker(ctx context.Context, tenantId uuid.UUID, opts *repository.CreateWorkerOpts) (*sqlcv1.Worker, error)
	AddWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)
	RemoveWorkerActions(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, actionIds []string) (int, error)
	UpsertWorkerLabels(ctx context.Context, workerId uuid.UUID, opts []repository.UpsertWorkerLabelOpts) ([]*sqlcv1.WorkerLabel, error)
	ActivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	DeactivateWorkerListener(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, sessionId uuid.UUID) (*sqlcv1.Worker, error)
	UpdateWorkerHeartbeat(ctx context.Context, tenantId uuid.UUID, workerId uuid.UUID, lastHeartbeatAt time.Time) error
}

// workflowPutter is the one admin service method the link calls.
type workflowPutter interface {
	PutWorkflow(ctx context.Context, req *v1.CreateWorkflowVersionRequest) (*v1.CreateWorkflowVersionResponse, error)
}

// Deps is what New needs from the engine.
type Deps struct {
	Dispatcher Dispatcher
	AdminV1    adminv1.AdminService
	Repo       repository.Repository
	Validator  validator.Validator
	Logger     *zerolog.Logger

	// DispatcherId pins every worker the link creates to the local dispatcher, since the
	// operator session that delivers its actions lives there.
	DispatcherId uuid.UUID

	// OperatorName is the v1_operator row name registrations attach their workers to; it is
	// the same name grpclink connects as. Defaults to "serverless".
	OperatorName string
}

// DefaultOperatorName matches grpclink's default so both links share one operator row per
// tenant.
const DefaultOperatorName = "serverless"

const (
	// tenantContextKey is the context key the dispatcher and admin services read the tenant
	// from; it must match the gRPC auth middleware's key.
	tenantContextKey = "tenant"

	// defaultSlotCount is the default slot config when the core passes none, as in
	// grpcoperator.
	defaultSlotCount = 100

	// maxActionsPerDelta caps the ids one AddWorkerActions or RemoveWorkerActions call carries,
	// the same chunk size the gRPC operator path applies per streamed delta.
	maxActionsPerDelta = 1000
)

// Link opens in-engine registrations.
type Link struct {
	dispatcher   Dispatcher
	admin        workflowPutter
	tenants      tenantStore
	operators    operatorStore
	workers      workerStore
	v            validator.Validator
	l            *zerolog.Logger
	dispatcherId uuid.UUID
	name         string

	// heartbeatInterval overrides the registration heartbeat cadence; zero means the default.
	// Tests shorten it.
	heartbeatInterval time.Duration
}

// New builds a Link. It panics on a nil dispatcher, admin service or repository, since the
// engine wires it once at startup and a nil there is a programming error.
func New(deps Deps) *Link {
	if deps.Dispatcher == nil || deps.AdminV1 == nil || deps.Repo == nil {
		panic("enginelink: dispatcher, admin service and repository are required")
	}

	l := deps.Logger

	if l == nil {
		nop := zerolog.Nop()
		l = &nop
	}

	v := deps.Validator

	if v == nil {
		v = validator.NewDefaultValidator()
	}

	name := deps.OperatorName

	if name == "" {
		name = DefaultOperatorName
	}

	return &Link{
		dispatcher:   deps.Dispatcher,
		admin:        deps.AdminV1,
		tenants:      deps.Repo.Tenant(),
		operators:    deps.Repo.Operators(),
		workers:      deps.Repo.Workers(),
		v:            v,
		l:            l,
		dispatcherId: deps.DispatcherId,
		name:         name,
	}
}

// ReleaseTenant implements link.TenantReleaser. The link holds no per-tenant state.
func (e *Link) ReleaseTenant(uuid.UUID) {}

// registerOpts carries the validation rules grpcoperator applies to a register request.
type registerOpts struct {
	Name    string   `validate:"required,hatchetName"`
	Actions []string `validate:"dive,actionId"`
}

// Open implements link.Link. It runs the registration steps of grpcoperator.Register and
// Listen in process: upsert the SERVERLESS operator row, create the worker with its slot
// config, link the initial action set in bulk chunks (the same path a streamed delta takes,
// not one upsert per action), write labels, activate the worker under a fresh listener
// session id, register the operator session with the dispatcher and notify the scheduler.
// The worker is activated only once its actions are linked. Heartbeats start with the
// registration and stop on Close.
func (e *Link) Open(ctx context.Context, tenantId uuid.UUID, opts link.OpenOpts) (link.Registration, error) {
	tenant, err := e.tenants.GetTenantByID(ctx, tenantId)

	if err != nil {
		return nil, fmt.Errorf("could not load tenant %s: %w", tenantId, err)
	}

	actions := unionActions(opts.Actions)

	if err := e.v.Validate(registerOpts{Name: e.name, Actions: actions}); err != nil {
		return nil, fmt.Errorf("invalid registration: %w", err)
	}

	tctx := withTenant(ctx, tenant)

	op, err := e.operators.UpsertServerlessOperator(ctx, tenantId, e.name)

	if err != nil {
		return nil, fmt.Errorf("could not upsert serverless operator %s: %w", e.name, err)
	}

	slotConfig := opts.SlotConfig

	if len(slotConfig) == 0 {
		slotConfig = map[string]int32{repository.SlotTypeDefault: defaultSlotCount}
	}

	operatorId := op.ID

	worker, err := e.workers.CreateNewWorker(ctx, tenantId, &repository.CreateWorkerOpts{
		DispatcherId: e.dispatcherId,
		Name:         workerName(e.dispatcherId),
		SlotConfig:   slotConfig,
		OperatorId:   &operatorId,
	})

	if err != nil {
		return nil, fmt.Errorf("could not create serverless worker: %w", err)
	}

	// The session id is the listener fence on the worker row: activation records it and the
	// deactivation on Close only succeeds while it is still the id on the row, so a newer
	// session on the same worker id is never marked inactive by an older one; see
	// grpcoperator.Listen.
	sessionId := uuid.New()

	l := e.l.With().
		Str("tenant_id", tenantId.String()).
		Str("operator_id", op.ID.String()).
		Str("worker_id", worker.ID.String()).
		Str("session_id", sessionId.String()).
		Logger()

	reg := newRegistration(e, tenant, worker.ID, sessionId, slotBuffer(slotConfig), &l)

	if err := reg.applyDelta(ctx, actions, e.workers.AddWorkerActions, "add"); err != nil {
		return nil, fmt.Errorf("could not link the initial actions of serverless worker %s: %w", worker.ID, err)
	}

	labels := labelOpts(opts.Labels)

	if _, err := e.workers.UpsertWorkerLabels(ctx, worker.ID, labels); err != nil {
		return nil, fmt.Errorf("could not upsert worker labels: %w", err)
	}

	if _, err := e.workers.ActivateWorkerListener(ctx, tenantId, worker.ID, sessionId); err != nil {
		return nil, fmt.Errorf("could not activate serverless worker %s: %w", worker.ID, err)
	}

	reg.release = e.dispatcher.AddOperatorSession(worker.ID, sessionId, sessionOperator{reg})

	e.dispatcher.NotifyNewWorker(tctx, tenant, worker.ID)

	reg.startHeartbeats()

	l.Info().Int("actions", len(actions)).Msg("serverless worker registered in engine")

	return reg, nil
}

// withTenant attaches the tenant under the key the dispatcher and admin services read.
func withTenant(ctx context.Context, tenant *sqlcv1.Tenant) context.Context {
	return context.WithValue(ctx, tenantContextKey, tenant) //nolint:staticcheck // key must match the gRPC auth middleware's
}

// workerName is "serverless-<dispatcher id>": one worker per tenant per process, whatever the
// number of the tenant's units the process owns.
func workerName(dispatcherId uuid.UUID) string {
	return fmt.Sprintf("serverless-%s", dispatcherId)
}

// slotBuffer sizes the registration's action buffer: the worker can never be assigned more
// tasks than it has slots, so a buffer that size only fills when the core stops reading.
func slotBuffer(slotConfig map[string]int32) int {
	total := 0

	for _, units := range slotConfig {
		if units > 0 {
			total += int(units)
		}
	}

	if total < 1 {
		total = 1
	}

	return total
}

// labelOpts converts the core's labels to repository label opts the way the SDK client maps
// them onto WorkerLabels: ints become int labels, everything else a string.
func labelOpts(labels map[string]interface{}) []repository.UpsertWorkerLabelOpts {
	out := make([]repository.UpsertWorkerLabelOpts, 0, len(labels))

	add := func(key string, value interface{}) {
		opt := repository.UpsertWorkerLabelOpts{Key: key}

		switch v := value.(type) {
		case int:
			iv := int32(v) // #nosec G115 -- label values are small operator-configured ints
			opt.IntValue = &iv
		case int32:
			iv := v
			opt.IntValue = &iv
		case int64:
			iv := int32(v) // #nosec G115 -- see above
			opt.IntValue = &iv
		case string:
			sv := v
			opt.StrValue = &sv
		default:
			sv := fmt.Sprintf("%v", v)
			opt.StrValue = &sv
		}

		out = append(out, opt)
	}

	for key, value := range labels {
		add(key, value)
	}

	return out
}

// workflowActions returns the action ids a worker must register to run every task of the
// workflow, normalized with types.ParseActionID as the admin service stores them, the same
// derivation the operator client applies in PutWorkflow.
func workflowActions(wf *v1.CreateWorkflowVersionRequest) ([]string, error) {
	if wf == nil {
		return nil, errors.New("workflow is required")
	}

	tasks := make([]*v1.CreateTaskOpts, 0, len(wf.Tasks)+1)
	tasks = append(tasks, wf.Tasks...)

	if wf.OnFailureTask != nil {
		tasks = append(tasks, wf.OnFailureTask)
	}

	actions := make([]string, 0, len(tasks))

	for i, task := range tasks {
		if task == nil {
			return nil, fmt.Errorf("workflow %s: task at index %d is nil", wf.Name, i)
		}

		if task.Action == "" {
			return nil, fmt.Errorf("workflow %s: task at index %d is missing required field 'Action'", wf.Name, i)
		}

		parsed, err := types.ParseActionID(task.Action)

		if err != nil {
			return nil, fmt.Errorf("workflow %s: %w", wf.Name, err)
		}

		actions = append(actions, parsed.String())
	}

	return actions, nil
}

// unionActions merges action lists in order, dropping duplicates and empty entries.
func unionActions(lists ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)

	for _, list := range lists {
		for _, action := range list {
			if strings.TrimSpace(action) == "" {
				continue
			}

			if _, ok := seen[action]; ok {
				continue
			}

			seen[action] = struct{}{}
			out = append(out, action)
		}
	}

	return out
}
