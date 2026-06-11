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
	"github.com/hatchet-dev/hatchet/pkg/operator/httpoperator"
	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type OperatorManager struct {
	repo              repository.Repository
	enc               encryption.EncryptionService
	taskEventWriter   operator.TaskEventWriter
	l                 *zerolog.Logger
	operatorsCh       chan operator.Operator
	donePollingCh     chan struct{}
	cleanup           func()
	operators         syncx.Map[uuid.UUID, operator.Operator]
	infraBlockedCIDRs []string
	dispatcherId      uuid.UUID
}

func NewOperatorManager(dispatcherId uuid.UUID, l *zerolog.Logger, repo repository.Repository, enc encryption.EncryptionService, infraBlockedCIDRs []string) *OperatorManager {
	om := &OperatorManager{
		dispatcherId:      dispatcherId,
		repo:              repo,
		l:                 l,
		enc:               enc,
		infraBlockedCIDRs: infraBlockedCIDRs,
		operatorsCh:       make(chan operator.Operator),
		donePollingCh:     make(chan struct{}, 1),
	}

	return om
}

// Start begins polling for operators. taskEventWriter is the sink operators use to report
// task results back through the dispatcher (the DispatcherImpl itself satisfies it); it is
// passed here rather than to the constructor because the dispatcher isn't fully built when
// the manager is created.
func (om *OperatorManager) Start(ctx context.Context, taskEventWriter operator.TaskEventWriter) <-chan operator.Operator {
	ctx, cancel := context.WithCancel(ctx)

	om.cleanup = cancel
	om.taskEventWriter = taskEventWriter

	go om.pollOperators(ctx)

	return om.operatorsCh
}

func (om *OperatorManager) Cleanup() {
	if om.cleanup != nil {
		om.cleanup()
	}

	<-om.donePollingCh

	// clean up the operators simultaneously
	wg := sync.WaitGroup{}

	om.operators.Range(func(key uuid.UUID, value operator.Operator) bool {
		wg.Go(value.Cleanup)
		return true
	})

	wg.Wait()

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

	for {
		select {
		case <-ctx.Done():
			om.donePollingCh <- struct{}{}
			return
		case <-t.C:
			operators, err := om.repo.Operators().ClaimOperators(ctx, om.dispatcherId)

			if err != nil {
				om.l.Error().Err(err).Msgf("could not poll operators")
			}

			om.setOperators(ctx, operators)
		}
	}
}

func (om *OperatorManager) setOperators(ctx context.Context, operators []*sqlcv1.V1Operator) {
	for _, op := range operators {
		var operator operator.Operator
		var written bool

		if op.Kind == sqlcv1.V1OperatorKindHTTPAPI {
			var ok bool
			operator, ok = om.operators.Load(op.ID)

			if !ok {
				// The slot config may vary per operator, so the operator type derives it
				// from its own config.
				slotConfig, err := httpoperator.SlotConfig(op)

				if err != nil {
					om.l.Error().Err(err).Msgf("could not determine slot config for http operator: %s", err.Error())
					continue
				}

				// Each operator instance gets its own freshly-created worker.
				worker, err := om.repo.Operators().CreateOperatorWorker(ctx, om.dispatcherId, op, slotConfig)

				if err != nil {
					om.l.Error().Err(err).Msgf("could not create worker for http operator: %s", err.Error())
					continue
				}

				operator, err = httpoperator.NewHTTPOperator(op, om.l, om.repo, om.taskEventWriter, om.enc, om.infraBlockedCIDRs, worker.ID)

				if err != nil {
					om.l.Error().Err(err).Msgf("could not construct http operator: %s", err.Error())
					continue
				}

				written = true
			}
		}

		if written {
			om.operators.Store(op.ID, operator)
			om.operatorsCh <- operator
		}
	}
}
