package olap

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	"github.com/hatchet-dev/hatchet/pkg/integrations/metrics/prometheus"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
)

type taskPrometheusUpdate struct {
	tenantId       string
	taskId         int64
	taskInsertedAt pgtype.Timestamptz
	readableStatus sqlcv1.V1ReadableStatusOlap
	workflowId     uuid.UUID
	isDAGTask      bool
}

type dagPrometheusUpdate struct {
	tenantId       string
	dagExternalId  uuid.UUID
	dagInsertedAt  pgtype.Timestamptz
	readableStatus sqlcv1.V1ReadableStatusOlap
	workflowId     uuid.UUID
}

func (o *OLAPControllerImpl) runTaskPrometheusUpdateWorker() {
	defer func() {
		if r := recover(); r != nil {
			_ = recoveryutils.RecoverWithAlert(o.l, o.a, r)
		}
	}()

	const batchSize = 1000
	batch := make([]taskPrometheusUpdate, 0, batchSize)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processBatch := func(updates []taskPrometheusUpdate) {
		if len(updates) == 0 {
			return
		}

		// Group by tenant
		tenantToUpdates := make(map[string][]taskPrometheusUpdate)
		for _, update := range updates {
			tenantToUpdates[update.tenantId] = append(tenantToUpdates[update.tenantId], update)
		}

		eg := errgroup.Group{}

		for tenantId, tenantUpdates := range tenantToUpdates {
			eg.Go(func() error {
				workflowIds := make([]uuid.UUID, 0, len(tenantUpdates))
				for _, update := range tenantUpdates {
					workflowIds = append(workflowIds, update.workflowId)
				}

				workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(o.taskPrometheusWorkerCtx, tenantId, workflowIds)
				if err != nil {
					return err
				}

				taskIds := make([]int64, 0, len(tenantUpdates))
				taskInsertedAts := make([]pgtype.Timestamptz, 0, len(tenantUpdates))
				readableStatuses := make([]sqlcv1.V1ReadableStatusOlap, 0, len(tenantUpdates))

				for _, update := range tenantUpdates {
					taskIds = append(taskIds, update.taskId)
					taskInsertedAts = append(taskInsertedAts, update.taskInsertedAt)
					readableStatuses = append(readableStatuses, update.readableStatus)
				}

				taskDurations, err := o.repo.OLAP().GetTaskDurationsByTaskIds(o.taskPrometheusWorkerCtx, tenantId, taskIds, taskInsertedAts, readableStatuses)
				if err != nil {
					return err
				}

				for _, update := range tenantUpdates {
					if update.isDAGTask {
						continue
					}

					workflowName := workflowNames[update.workflowId]
					if workflowName == "" {
						continue
					}

					taskDuration := taskDurations[update.taskId]
					if taskDuration == nil || !taskDuration.StartedAt.Valid || !taskDuration.FinishedAt.Valid {
						continue
					}

					duration := int(taskDuration.FinishedAt.Time.Sub(taskDuration.StartedAt.Time).Milliseconds())
					prometheus.TenantWorkflowDurationBuckets.WithLabelValues(tenantId, workflowName, string(update.readableStatus)).Observe(float64(duration))
				}

				return nil
			})
		}

		err := eg.Wait()

		if err != nil {
			o.l.Error().Err(err).Msg("failed to process task prometheus updates")
		}
	}

	for {
		select {
		case <-o.taskPrometheusWorkerCtx.Done():
			// Process remaining batch before exiting
			processBatch(batch)
			return
		case update, ok := <-o.taskPrometheusUpdateCh:
			if !ok {
				// Channel closed, process remaining batch
				processBatch(batch)
				return
			}
			batch = append(batch, update)
			if len(batch) >= batchSize {
				processBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// Process batch on timer
			processBatch(batch)
			batch = batch[:0]
		}
	}
}

func (o *OLAPControllerImpl) runDAGPrometheusUpdateWorker() {
	defer func() {
		if r := recover(); r != nil {
			_ = recoveryutils.RecoverWithAlert(o.l, o.a, r)
		}
	}()

	const batchSize = 1000
	batch := make([]dagPrometheusUpdate, 0, batchSize)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	processBatch := func(updates []dagPrometheusUpdate) {
		if len(updates) == 0 {
			return
		}

		// Group by tenant
		tenantToUpdates := make(map[string][]dagPrometheusUpdate)
		for _, update := range updates {
			tenantToUpdates[update.tenantId] = append(tenantToUpdates[update.tenantId], update)
		}

		eg := errgroup.Group{}

		for tenantId, tenantUpdates := range tenantToUpdates {
			eg.Go(func() error {
				workflowIds := make([]uuid.UUID, 0, len(tenantUpdates))
				for _, update := range tenantUpdates {
					workflowIds = append(workflowIds, update.workflowId)
				}

				workflowNames, err := o.repo.Workflows().ListWorkflowNamesByIds(o.dagPrometheusWorkerCtx, tenantId, workflowIds)
				if err != nil {
					return err
				}

				dagExternalIds := make([]uuid.UUID, 0, len(tenantUpdates))
				var minInsertedAt pgtype.Timestamptz

				for _, update := range tenantUpdates {
					dagExternalIds = append(dagExternalIds, update.dagExternalId)

					if !minInsertedAt.Valid || update.dagInsertedAt.Time.Before(minInsertedAt.Time) {
						minInsertedAt = update.dagInsertedAt
					}
				}

				dagDurations, err := o.repo.OLAP().GetDAGDurations(o.dagPrometheusWorkerCtx, tenantId, dagExternalIds, minInsertedAt)
				if err != nil {
					return err
				}

				for _, update := range tenantUpdates {
					workflowName := workflowNames[update.workflowId]
					if workflowName == "" {
						continue
					}

					dagDuration := dagDurations[update.dagExternalId.String()]
					if dagDuration == nil || !dagDuration.StartedAt.Valid || !dagDuration.FinishedAt.Valid {
						continue
					}

					duration := int(dagDuration.FinishedAt.Time.Sub(dagDuration.StartedAt.Time).Milliseconds())
					prometheus.TenantWorkflowDurationBuckets.WithLabelValues(tenantId, workflowName, string(update.readableStatus)).Observe(float64(duration))
				}

				return nil
			})
		}

		err := eg.Wait()

		if err != nil {
			o.l.Error().Err(err).Msg("failed to process dag prometheus updates")
		}
	}

	for {
		select {
		case <-o.dagPrometheusWorkerCtx.Done():
			// Process remaining batch before exiting
			processBatch(batch)
			return
		case update, ok := <-o.dagPrometheusUpdateCh:
			if !ok {
				// Channel closed, process remaining batch
				processBatch(batch)
				return
			}
			batch = append(batch, update)
			if len(batch) >= batchSize {
				processBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			// Process batch on timer
			processBatch(batch)
			batch = batch[:0]
		}
	}
}
