package manager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/syncx"
	"github.com/hatchet-dev/hatchet/pkg/encryption"
	"github.com/hatchet-dev/hatchet/pkg/operator"
	"github.com/hatchet-dev/hatchet/pkg/operator/dagoperator"
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

// heartbeatTimeout bounds a single bulk heartbeat write. Heartbeats use a detached context
// because they must keep running while operators drain during shutdown.
const heartbeatTimeout = 5 * time.Second

// bulkPauseTimeout bounds the single statement that pauses all operator workers during
// manager shutdown.
const bulkPauseTimeout = 30 * time.Second

type OperatorManager struct {
	repo            repository.Repository
	enc             encryption.EncryptionService
	taskEventWriter operator.TaskEventWriter
	l               *zerolog.Logger
	operatorsCh     chan []operator.Operator
	donePollingCh   chan struct{}
	doneHeartbeatCh chan struct{}
	cleanup         func()
	stopHeartbeats  func()
	operators       syncx.Map[uuid.UUID, operator.Operator]

	// draining holds worker IDs of operators that are draining in-flight tasks; they must
	// keep receiving heartbeats until the drain completes. mu guards it (poll loop adds,
	// per-operator drain goroutines remove, heartbeat loop reads).
	draining map[uuid.UUID]struct{}

	infraBlockedCIDRs []string
	dispatcherId      uuid.UUID
	mu                sync.Mutex

	// drains tracks in-flight per-operator drain goroutines so Cleanup can await them.
	drains sync.WaitGroup
}

func NewOperatorManager(dispatcherId uuid.UUID, l *zerolog.Logger, repo repository.Repository, enc encryption.EncryptionService, infraBlockedCIDRs []string) *OperatorManager {
	om := &OperatorManager{
		dispatcherId:      dispatcherId,
		repo:              repo,
		l:                 l,
		enc:               enc,
		infraBlockedCIDRs: infraBlockedCIDRs,
		operatorsCh:       make(chan []operator.Operator),
		donePollingCh:     make(chan struct{}, 1),
		doneHeartbeatCh:   make(chan struct{}),
		draining:          make(map[uuid.UUID]struct{}),
	}

	return om
}

// Start begins polling for operators. taskEventWriter is the sink operators use to report
// task results back through the dispatcher (the DispatcherImpl itself satisfies it); it is
// passed here rather than to the constructor because the dispatcher isn't fully built when
// the manager is created.
//
// The returned channel carries the full set of active operators on every poll; consumers
// should treat each message as the desired state and remove anything no longer present.
func (om *OperatorManager) Start(ctx context.Context, taskEventWriter operator.TaskEventWriter) <-chan []operator.Operator {
	ctx, cancel := context.WithCancel(ctx)

	om.cleanup = cancel
	om.taskEventWriter = taskEventWriter

	// heartbeats deliberately outlive the polling context: they must keep running while
	// operators drain during shutdown, and are stopped explicitly at the end of Cleanup
	heartbeatCtx, stopHeartbeats := context.WithCancel(context.WithoutCancel(ctx))

	om.stopHeartbeats = stopHeartbeats

	go om.pollOperators(ctx)
	go om.runHeartbeats(heartbeatCtx)

	return om.operatorsCh
}

func (om *OperatorManager) Cleanup() {
	if om.cleanup == nil {
		// Start was never called; there is nothing to drain.
		close(om.operatorsCh)
		return
	}

	om.cleanup()

	<-om.donePollingCh

	// pause every operator worker in a single statement (rather than one update per
	// operator) so the scheduler stops assigning to them while they drain
	workerIds := make([]uuid.UUID, 0)

	om.operators.Range(func(_ uuid.UUID, op operator.Operator) bool {
		workerIds = append(workerIds, op.WorkerId())
		return true
	})

	if len(workerIds) > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), bulkPauseTimeout)

		if err := om.repo.Workers().PauseWorkers(ctx, workerIds); err != nil {
			om.l.Error().Err(err).Msgf("could not pause operator workers on shutdown")
		}

		cancel()
	}

	// drain the operators simultaneously; Drain skips the per-worker pause done in bulk above
	wg := sync.WaitGroup{}

	om.operators.Range(func(key uuid.UUID, value operator.Operator) bool {
		wg.Go(value.Drain)
		return true
	})

	wg.Wait()

	// wait for any individually-removed operators that are still draining
	om.drains.Wait()

	// stop heartbeats only after every operator has finished draining, so workers stay
	// registered until their in-flight tasks complete
	om.stopHeartbeats()
	<-om.doneHeartbeatCh

	// close the channel
	close(om.operatorsCh)
}

func (om *OperatorManager) HandleAction(ctx context.Context, operatorId uuid.UUID, action *contracts.AssignedAction) error {
	op, ok := om.operators.Load(operatorId)

	if !ok {
		return fmt.Errorf("could not send action to operator worker: not found in operator map")
	}

	return op.HandleAction(ctx, action)
}

func (om *OperatorManager) pollOperators(ctx context.Context) {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			om.donePollingCh <- struct{}{}
			return
		case <-t.C:
			operators, err := om.repo.Operators().ClaimOperators(ctx, om.dispatcherId)

			if err != nil {
				// skip reconciliation entirely: a failed poll says nothing about which
				// operators are still claimed, so it must not count toward teardown misses
				om.l.Error().Err(err).Msgf("could not poll operators")
				continue
			}

			om.reconcileOperators(ctx, operators)
		}
	}
}

// reconcileOperators treats the claim result as the desired state: it instantiates operators
// which are newly claimed, tears down operators no longer in the claim result (an operator
// assigned to this dispatcher cannot be claimed by another active dispatcher, so an absence
// means it was deleted or this dispatcher lost it), and reports the resulting full active
// set on operatorsCh.
func (om *OperatorManager) reconcileOperators(ctx context.Context, claimed []*sqlcv1.V1Operator) {
	claimedIds := make(map[uuid.UUID]struct{}, len(claimed))

	for _, op := range claimed {
		claimedIds[op.ID] = struct{}{}

		if _, ok := om.operators.Load(op.ID); ok {
			continue
		}

		newOperator := om.instantiateOperator(ctx, op)

		if newOperator != nil {
			om.operators.Store(op.ID, newOperator)
		}
	}

	om.operators.Range(func(id uuid.UUID, op operator.Operator) bool {
		if _, ok := claimedIds[id]; ok {
			return true
		}

		om.teardownOperator(id, op)

		return true
	})

	// report the full active set; the dispatcher reconciles its routing table against it.
	// the resend on every poll (even when unchanged) is what lets the dispatcher self-heal.
	select {
	case om.operatorsCh <- om.activeOperators():
	case <-ctx.Done():
	}
}

// instantiateOperator constructs the operator for a newly-claimed row, creating a fresh
// worker for this instance. Returns nil (after logging) if the operator could not be built.
func (om *OperatorManager) instantiateOperator(ctx context.Context, op *sqlcv1.V1Operator) operator.Operator {
	// The slot config may vary per operator, so each operator type derives it from its own
	// config.
	slotConfig, err := slotConfigForKind(op)

	if err != nil {
		om.l.Error().Err(err).Msgf("could not determine slot config for operator: %s", err.Error())
		return nil
	}

	if slotConfig == nil {
		// Unsupported kind; nothing to instantiate.
		return nil
	}

	// Each operator instance gets its own freshly-created worker.
	worker, err := om.repo.Operators().CreateOperatorWorker(ctx, om.dispatcherId, op, slotConfig)

	if err != nil {
		om.l.Error().Err(err).Msgf("could not create worker for operator: %s", err.Error())
		return nil
	}

	switch op.Kind {
	case sqlcv1.V1OperatorKindHTTPAPI:
		newOperator, err := httpoperator.NewHTTPOperator(op, om.l, om.repo, om.taskEventWriter, om.enc, om.infraBlockedCIDRs, worker.ID)

		if err != nil {
			om.l.Error().Err(err).Msgf("could not construct http operator: %s", err.Error())
			return nil
		}

		return newOperator
	case sqlcv1.V1OperatorKindDAG:
		newOperator, err := dagoperator.NewDAGOperator(op, om.l, om.repo, om.taskEventWriter, worker.ID)

		if err != nil {
			om.l.Error().Err(err).Msgf("could not construct dag operator: %s", err.Error())
			return nil
		}

		return newOperator
	default:
		return nil
	}
}

// slotConfigForKind derives the worker slot config for an operator from its kind-specific
// config. It returns (nil, nil) for unsupported kinds so the caller skips instantiation.
func slotConfigForKind(op *sqlcv1.V1Operator) (map[string]int32, error) {
	switch op.Kind {
	case sqlcv1.V1OperatorKindHTTPAPI:
		return httpoperator.SlotConfig(op)
	case sqlcv1.V1OperatorKindDAG:
		return dagoperator.SlotConfig(op)
	default:
		return nil, nil
	}
}

// teardownOperator removes an operator that is no longer claimed by this dispatcher and
// drains it in the background. The worker is registered as draining before the operator is
// removed from the map so the heartbeat loop never sees a gap while tasks are in flight.
func (om *OperatorManager) teardownOperator(id uuid.UUID, op operator.Operator) {
	workerId := op.WorkerId()

	om.mu.Lock()
	om.draining[workerId] = struct{}{}
	om.mu.Unlock()

	om.operators.Delete(id)

	om.l.Info().Msgf("operator %s is no longer claimed by this dispatcher, draining worker %s", id, workerId)

	om.drains.Go(func() {
		op.Cleanup()

		om.mu.Lock()
		delete(om.draining, workerId)
		om.mu.Unlock()
	})
}

func (om *OperatorManager) activeOperators() []operator.Operator {
	active := make([]operator.Operator, 0)

	om.operators.Range(func(_ uuid.UUID, op operator.Operator) bool {
		active = append(active, op)
		return true
	})

	return active
}

// runHeartbeats updates the heartbeat for every operator worker (active and draining) in a
// single statement per tick. Its context is detached from the polling context's
// cancellation because heartbeats must continue while operators drain during shutdown; it
// is cancelled explicitly at the end of Cleanup.
func (om *OperatorManager) runHeartbeats(ctx context.Context) {
	t := time.NewTicker(4 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			close(om.doneHeartbeatCh)
			return
		case <-t.C:
			workerIds := om.heartbeatWorkerIds()

			if len(workerIds) == 0 {
				continue
			}

			tickCtx, cancel := context.WithTimeout(ctx, heartbeatTimeout)

			err := om.repo.Workers().UpdateWorkerHeartbeats(tickCtx, workerIds, time.Now().UTC())

			cancel()

			if err != nil {
				om.l.Error().Err(err).Msgf("could not update operator worker heartbeats")
			}
		}
	}
}

func (om *OperatorManager) heartbeatWorkerIds() []uuid.UUID {
	seen := make(map[uuid.UUID]struct{})

	om.operators.Range(func(_ uuid.UUID, op operator.Operator) bool {
		seen[op.WorkerId()] = struct{}{}
		return true
	})

	om.mu.Lock()
	for workerId := range om.draining {
		seen[workerId] = struct{}{}
	}
	om.mu.Unlock()

	workerIds := make([]uuid.UUID, 0, len(seen))

	for workerId := range seen {
		workerIds = append(workerIds, workerId)
	}

	return workerIds
}
