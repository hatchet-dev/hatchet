package operatorsvc_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/operatorsvc"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// grpcRegisterOpts is what a wire registration passes: a self-leased GRPC row whose workers
// are metered.
func grpcRegisterOpts(name string) operatorsvc.RegisterOpts {
	return operatorsvc.RegisterOpts{Name: name, Kind: sqlcv1.V1OperatorKindGRPC, Leasing: sqlcv1.V1OperatorLeasingSELF}
}

func TestRegisterCreatesWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	label := "gpu"

	opts := grpcRegisterOpts("my-operator")
	opts.SlotConfig = map[string]int32{"default": 5}
	opts.Labels = map[string]*contracts.WorkerLabels{"kind": {StrValue: &label}}

	reg, err := svc.Register(t.Context(), tenant, opts)

	require.NoError(t, err)
	assert.Equal(t, tenant.ID, reg.TenantId)
	assert.False(t, reg.Resumed)

	op, err := svc.operators.GetOperatorById(t.Context(), reg.OperatorId)
	require.NoError(t, err)
	assert.Equal(t, "my-operator", op.Name)
	assert.Equal(t, sqlcv1.V1OperatorKindGRPC, op.Kind)
	assert.Equal(t, sqlcv1.V1OperatorLeasingSELF, op.Leasing)
	assert.Equal(t, op, reg.Operator)

	created := svc.workers.Created()
	require.Len(t, created, 1)
	assert.Equal(t, op.Name, created[0].Name)
	assert.Equal(t, svc.dispatcherId, created[0].DispatcherId)
	require.NotNil(t, created[0].OperatorId)
	assert.Equal(t, op.ID, *created[0].OperatorId)
	assert.Empty(t, created[0].Actions, "registration never registers actions")
	assert.Equal(t, map[string]int32{"default": 5}, created[0].SlotConfig)
	assert.False(t, created[0].ExemptFromLimits, "a worker is metered unless the caller exempts it")

	require.Len(t, svc.workers.Labels(reg.WorkerId), 1)
	assert.Equal(t, "kind", svc.workers.Labels(reg.WorkerId)[0].Key)

	// a second registration of the same name reuses the operator and creates another worker
	again, err := svc.Register(t.Context(), tenant, grpcRegisterOpts("my-operator"))
	require.NoError(t, err)
	assert.Equal(t, reg.OperatorId, again.OperatorId)
	assert.NotEqual(t, reg.WorkerId, again.WorkerId)
	assert.False(t, again.Resumed)
	assert.Equal(t, map[string]int32{repository.SlotTypeDefault: int32(100)}, svc.workers.Created()[1].SlotConfig, "empty slot config defaults")
}

// The worker is named after the operator unless the caller names it, so replicas of one
// operator can be told apart.
func TestRegisterNamesWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	opts := grpcRegisterOpts("my-operator")
	opts.WorkerName = "my-operator-replica-2"

	_, err := svc.Register(t.Context(), tenant, opts)
	require.NoError(t, err)

	require.Len(t, svc.workers.Created(), 1)
	assert.Equal(t, "my-operator-replica-2", svc.workers.Created()[0].Name)
}

func TestRegisterResumesOwnWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	first, err := svc.Register(t.Context(), tenant, grpcRegisterOpts("my-operator"))
	require.NoError(t, err)

	opts := grpcRegisterOpts("my-operator")
	opts.ResumeWorkerId = &first.WorkerId

	resumed, err := svc.Register(t.Context(), tenant, opts)
	require.NoError(t, err)
	assert.True(t, resumed.Resumed)
	assert.Equal(t, first.WorkerId, resumed.WorkerId)
	assert.Equal(t, first.OperatorId, resumed.OperatorId)
	assert.Len(t, svc.workers.Created(), 1, "resume must not create a worker")
}

// An operator that paused its worker to drain, and then crashed, would otherwise come back to a
// worker the scheduler never assigns to.
func TestRegisterResumeClearsThePause(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	first, err := svc.Register(t.Context(), tenant, grpcRegisterOpts("my-operator"))
	require.NoError(t, err)

	require.NoError(t, svc.PauseWorker(t.Context(), tenant, first.WorkerId, true))
	require.True(t, svc.workers.IsPaused(first.WorkerId))

	opts := grpcRegisterOpts("my-operator")
	opts.ResumeWorkerId = &first.WorkerId

	resumed, err := svc.Register(t.Context(), tenant, opts)
	require.NoError(t, err)
	require.True(t, resumed.Resumed)
	assert.False(t, svc.workers.IsPaused(first.WorkerId), "a resumed worker is assignable again")
}

func TestRegisterDoesNotResumeAnotherOperatorsWorker(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	other, err := svc.Register(t.Context(), tenant, grpcRegisterOpts("other-operator"))
	require.NoError(t, err)

	unknown := uuid.New()

	cases := []struct {
		name     string
		workerId uuid.UUID
	}{
		{name: "another operator's worker", workerId: other.WorkerId},
		{name: "unknown worker", workerId: unknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			before := len(svc.workers.Created())

			opts := grpcRegisterOpts("my-operator")
			opts.ResumeWorkerId = &tc.workerId

			reg, err := svc.Register(t.Context(), tenant, opts)
			require.NoError(t, err)
			assert.False(t, reg.Resumed)
			assert.NotEqual(t, tc.workerId, reg.WorkerId)
			assert.Len(t, svc.workers.Created(), before+1, "a worker that cannot be resumed is replaced by a new one")
		})
	}
}

func TestRegisterRejects(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}

	t.Run("missing tenant", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(t.Context(), nil, grpcRegisterOpts("op"))
		assert.Equal(t, codes.Unauthenticated, status.Code(err))
	})

	for _, name := range []string{"", "bad name!"} {
		t.Run("invalid name "+name, func(t *testing.T) {
			svc := newTestService(t, nil)
			_, err := svc.Register(t.Context(), tenant, grpcRegisterOpts(name))
			assert.Equal(t, codes.InvalidArgument, status.Code(err))
			assert.Zero(t, svc.operators.Count(), "validation runs before the upsert")
		})
	}

	// only contract operators are upserted through a session; the DAG operator's rows are the
	// engine's own and are registered by the id of their claimed row
	t.Run("unsupported kind", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{Name: "op", Kind: sqlcv1.V1OperatorKindDAG, Leasing: sqlcv1.V1OperatorLeasingSELF})
		require.Error(t, err)
		assert.Zero(t, svc.operators.Count())
	})

	// a named registration says who keeps the row alive; it cannot leave that unsaid
	t.Run("missing leasing", func(t *testing.T) {
		svc := newTestService(t, nil)
		_, err := svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{Name: "op", Kind: sqlcv1.V1OperatorKindGRPC})
		require.Error(t, err)
		assert.Zero(t, svc.operators.Count())
	})
}

// The leasing a registration names is written to the row whether the upsert creates or finds
// it, so a row the engine was leasing that registers itself leaves the claim set, and the other
// way round.
func TestRegisterSetsLeasing(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	managed := grpcRegisterOpts("op")
	managed.Leasing = sqlcv1.V1OperatorLeasingMANAGED

	reg, err := svc.Register(t.Context(), tenant, managed)
	require.NoError(t, err)
	assert.Equal(t, sqlcv1.V1OperatorLeasingMANAGED, reg.Operator.Leasing)

	again, err := svc.Register(t.Context(), tenant, grpcRegisterOpts("op"))
	require.NoError(t, err)
	assert.Equal(t, reg.OperatorId, again.OperatorId, "the same (tenant, name, kind) row")
	assert.Equal(t, sqlcv1.V1OperatorLeasingSELF, again.Operator.Leasing, "a repeat registration takes the leasing it names")
}

// Limit exemption is the caller's to grant: the in-process host grants it to every worker it
// creates, whether the row is claimed or upserted.
func TestRegisterExemptsWorkerOnRequest(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	exempt := grpcRegisterOpts("op")
	exempt.ExemptFromLimits = true

	_, err := svc.Register(t.Context(), tenant, exempt)
	require.NoError(t, err)

	row := svc.operators.Put(&sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "dag", Kind: sqlcv1.V1OperatorKindDAG, Leasing: sqlcv1.V1OperatorLeasingMANAGED})

	_, err = svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{OperatorId: &row.ID, ExemptFromLimits: true})
	require.NoError(t, err)

	created := svc.workers.Created()
	require.Len(t, created, 2)
	assert.True(t, created[0].ExemptFromLimits)
	assert.True(t, created[1].ExemptFromLimits)
}

// A claimed row is registered by id: nothing is upserted, the worker is named after the row and
// the row is pointed at the worker so ClaimOperators keeps seeing the assignment.
func TestRegisterClaimedRow(t *testing.T) {
	tenant := &sqlcv1.Tenant{ID: uuid.New()}
	svc := newTestService(t, nil)

	row := svc.operators.Put(&sqlcv1.V1Operator{ID: uuid.New(), TenantID: tenant.ID, Name: "dag", Kind: sqlcv1.V1OperatorKindDAG})

	reg, err := svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{OperatorId: &row.ID, SlotConfig: map[string]int32{"durable": 3}})
	require.NoError(t, err)

	assert.Equal(t, row.ID, reg.OperatorId)
	assert.Equal(t, 1, svc.operators.Count(), "no row is upserted")

	created := svc.workers.Created()
	require.Len(t, created, 1)
	assert.Equal(t, "dag", created[0].Name)
	assert.Equal(t, map[string]int32{"durable": 3}, created[0].SlotConfig)

	op, err := svc.operators.GetOperatorById(t.Context(), row.ID)
	require.NoError(t, err)
	require.NotNil(t, op.WorkerID)
	assert.Equal(t, reg.WorkerId, *op.WorkerID, "the row points at the session's worker")

	// a row of another tenant, or no row at all, is refused the same way
	other := svc.operators.Put(&sqlcv1.V1Operator{ID: uuid.New(), TenantID: uuid.New(), Name: "dag", Kind: sqlcv1.V1OperatorKindDAG})

	_, err = svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{OperatorId: &other.ID})
	assert.Equal(t, codes.NotFound, status.Code(err))

	missing := uuid.New()
	_, err = svc.Register(t.Context(), tenant, operatorsvc.RegisterOpts{OperatorId: &missing})
	assert.Equal(t, codes.NotFound, status.Code(err))
}
