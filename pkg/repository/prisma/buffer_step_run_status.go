package prisma

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func newBulkStepRunStatusBuffer(shared *sharedRepository) (*buffer.TenantBufferManager[*updateStepRunQueueData, pgtype.UUID], error) {
	statusBufOpts := buffer.TenantBufManagerOpts[*updateStepRunQueueData, pgtype.UUID]{
		Name:       "update_step_run_status",
		OutputFunc: shared.bulkUpdateStepRunStatuses,
		SizeFunc:   sizeOfUpdateData,
		L:          shared.l,
		V:          shared.v,
	}

	var err error
	manager, err := buffer.NewTenantBufManager(statusBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func (s *sharedRepository) bulkUpdateStepRunStatuses(ctx context.Context, opts []*updateStepRunQueueData) ([]*pgtype.UUID, error) {
	stepRunIds := make([]*pgtype.UUID, 0, len(opts))

	eventTimeSeen := make([]time.Time, 0, len(opts))
	eventReasons := make([]dbsqlc.StepRunEventReason, 0, len(opts))
	eventStepRunIds := make([]pgtype.UUID, 0, len(opts))
	eventTenantIds := make([]string, 0, len(opts))
	eventSeverities := make([]dbsqlc.StepRunEventSeverity, 0, len(opts))
	eventMessages := make([]string, 0, len(opts))
	eventData := make([]map[string]interface{}, 0, len(opts))

	for _, item := range opts {
		stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)
		stepRunIds = append(stepRunIds, &stepRunId)

		if item.Status == nil {
			continue
		}

		switch dbsqlc.StepRunStatus(*item.Status) {
		case dbsqlc.StepRunStatusRUNNING:
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventTimeSeen = append(eventTimeSeen, *item.StartedAt)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonSTARTED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run started at %s", item.StartedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusFAILED:
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)

			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventMessage := fmt.Sprintf("Step run failed on %s", item.FinishedAt.Format(time.RFC1123))
			eventReason := dbsqlc.StepRunEventReasonFAILED

			if item.Error != nil && *item.Error == "TIMED_OUT" {
				eventReason = dbsqlc.StepRunEventReasonTIMEDOUT
				eventMessage = "Step exceeded timeout duration"
			}

			eventReasons = append(eventReasons, eventReason)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityCRITICAL)
			eventMessages = append(eventMessages, eventMessage)
			eventData = append(eventData, map[string]interface{}{
				"retry_count": item.RetryCount,
			})
		case dbsqlc.StepRunStatusCANCELLED:
			eventTimeSeen = append(eventTimeSeen, *item.CancelledAt)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonCANCELLED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityWARNING)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run was cancelled on %s for the following reason: %s", item.CancelledAt.Format(time.RFC1123), *item.CancelledReason))
			eventData = append(eventData, map[string]interface{}{})
		case dbsqlc.StepRunStatusSUCCEEDED:
			eventTimeSeen = append(eventTimeSeen, *item.FinishedAt)
			eventStepRunIds = append(eventStepRunIds, stepRunId)
			eventTenantIds = append(eventTenantIds, item.TenantId)
			eventReasons = append(eventReasons, dbsqlc.StepRunEventReasonFINISHED)
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
			eventMessages = append(eventMessages, fmt.Sprintf("Step run finished at %s", item.FinishedAt.Format(time.RFC1123)))
			eventData = append(eventData, map[string]interface{}{})
		}
	}

	eg := errgroup.Group{}

	if len(opts) > 0 {
		eg.Go(func() error {
			insertInternalQITenantIds := make([]pgtype.UUID, 0, len(opts))
			insertInternalQIQueues := make([]dbsqlc.InternalQueue, 0, len(opts))
			insertInternalQIData := make([]any, 0, len(opts))

			for _, item := range opts {
				if item.Status == nil {
					continue
				}

				itemCp := item

				insertInternalQITenantIds = append(insertInternalQITenantIds, sqlchelpers.UUIDFromStr(itemCp.TenantId))
				insertInternalQIQueues = append(insertInternalQIQueues, dbsqlc.InternalQueueSTEPRUNUPDATEV2)
				insertInternalQIData = append(insertInternalQIData, itemCp)
			}

			err := bulkInsertInternalQueueItem(
				ctx,
				s.pool,
				s.queries,
				insertInternalQITenantIds,
				insertInternalQIQueues,
				insertInternalQIData,
			)

			if err != nil {
				return err
			}

			return nil
		})
	}

	if len(eventStepRunIds) > 0 {
		for i, stepRunId := range eventStepRunIds {
			err := s.bulkEventBuffer.FireForget(eventTenantIds[i], &repository.CreateStepRunEventOpts{
				StepRunId:     sqlchelpers.UUIDToStr(stepRunId),
				EventMessage:  &eventMessages[i],
				EventReason:   &eventReasons[i],
				EventSeverity: &eventSeverities[i],
				Timestamp:     &eventTimeSeen[i],
				EventData:     eventData[i],
			})

			if err != nil {
				s.l.Err(err).Msg("could not buffer step run event")
			}
		}
	}

	err := eg.Wait()

	if err != nil {
		return nil, err
	}

	return stepRunIds, nil
}
