package prisma

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/buffer"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/dbsqlc"
	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/sqlchelpers"
)

func newBulkEventWriter(shared *sharedRepository, conf buffer.ConfigFileBuffer) (*buffer.TenantBufferManager[*repository.CreateStepRunEventOpts, int], error) {
	eventBufOpts := buffer.TenantBufManagerOpts[*repository.CreateStepRunEventOpts, int]{
		Name:     "step_run_event_buffer",
		SizeFunc: sizeOfEventData,
		L:        shared.l,
		V:        shared.v,
		Config:   conf,
	}

	if conf.SerialBuffer {
		shared.l.Warn().Msg("using serial buffer for step run events")
		eventBufOpts.OutputFunc = shared.serialWriteStepRunEvent
	} else {
		eventBufOpts.OutputFunc = shared.bulkWriteStepRunEvents
	}

	manager, err := buffer.NewTenantBufManager(eventBufOpts)

	if err != nil {
		shared.l.Err(err).Msg("could not create tenant buffer manager")
		return nil, err
	}

	return manager, nil
}

func sizeOfEventData(item *repository.CreateStepRunEventOpts) int {
	size := len(*item.EventMessage)

	for k, v := range item.EventData {
		size += len(k) + len(fmt.Sprintf("%v", v))
	}

	return size
}

func sortByStepRunId(opts []*repository.CreateStepRunEventOpts) []*repository.CreateStepRunEventOpts {

	sort.SliceStable(opts, func(i, j int) bool {
		return opts[i].StepRunId < opts[j].StepRunId
	})

	return opts
}

func (w *sharedRepository) serialWriteStepRunEvent(ctx context.Context, opts []*repository.CreateStepRunEventOpts) ([]*int, error) {
	res := make([]*int, 0, len(opts))

	for i, item := range opts {
		index := i
		res = append(res, &index)
		var eventData []byte
		var err error

		if item.EventData != nil {
			eventData, err = json.Marshal(item.EventData)

			if err != nil {
				return nil, fmt.Errorf("could not marshal step run event data: %w", err)
			}
		}

		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 1000)
		if err != nil {
			return nil, fmt.Errorf("could not prepare transaction: %w", err)
		}
		defer rollback()
		err = w.queries.CreateStepRunEvent(ctx, tx, dbsqlc.CreateStepRunEventParams{
			Steprunid: sqlchelpers.UUIDFromStr(item.StepRunId),
			Reason:    *item.EventReason,
			Severity:  *item.EventSeverity,
			Message:   *item.EventMessage,
			Data:      eventData,
		})

		if err != nil {
			return nil, fmt.Errorf("could not create step run event: %w", err)
		}
		err = commit(ctx)
		if err != nil {
			return nil, fmt.Errorf("could not commit transaction: %w", err)
		}

	}

	return res, nil

}

func (w *sharedRepository) bulkWriteStepRunEvents(ctx context.Context, opts []*repository.CreateStepRunEventOpts) ([]*int, error) {
	res := make([]*int, 0, len(opts))
	eventTimeSeen := make([]pgtype.Timestamp, 0, len(opts))
	eventReasons := make([]dbsqlc.StepRunEventReason, 0, len(opts))
	eventStepRunIds := make([]pgtype.UUID, 0, len(opts))
	eventSeverities := make([]dbsqlc.StepRunEventSeverity, 0, len(opts))
	eventMessages := make([]string, 0, len(opts))
	eventData := make([]map[string]interface{}, 0, len(opts))
	dedupe := make(map[string]bool)

	orderedOpts := sortByStepRunId(opts)

	for i, item := range orderedOpts {
		index := i
		res = append(res, &index)

		if item.EventMessage == nil || item.EventReason == nil || item.StepRunId == "" {
			continue
		}

		stepRunId := sqlchelpers.UUIDFromStr(item.StepRunId)
		dedupeKey := fmt.Sprintf("EVENT-%s-%s", item.StepRunId, *item.EventReason)

		if _, ok := dedupe[dedupeKey]; ok {
			continue
		}

		dedupe[dedupeKey] = true

		eventStepRunIds = append(eventStepRunIds, stepRunId)
		eventMessages = append(eventMessages, *item.EventMessage)
		eventReasons = append(eventReasons, *item.EventReason)

		if item.EventSeverity != nil {
			eventSeverities = append(eventSeverities, *item.EventSeverity)
		} else {
			eventSeverities = append(eventSeverities, dbsqlc.StepRunEventSeverityINFO)
		}

		if item.EventData != nil {
			eventData = append(eventData, item.EventData)
		} else {
			eventData = append(eventData, map[string]interface{}{})
		}

		if item.Timestamp != nil {
			eventTimeSeen = append(eventTimeSeen, sqlchelpers.TimestampFromTime(*item.Timestamp))
		} else {
			eventTimeSeen = append(eventTimeSeen, sqlchelpers.TimestampFromTime(time.Now().UTC()))
		}
	}

	err := sqlchelpers.DeadlockRetry(w.l, func() (err error) {
		tx, commit, rollback, err := sqlchelpers.PrepareTx(ctx, w.pool, w.l, 10000)

		if err != nil {
			return err
		}

		defer rollback()

		err = bulkStepRunEvents(
			ctx,
			w.l,
			tx,
			w.queries,
			eventStepRunIds,
			eventTimeSeen,
			eventReasons,
			eventSeverities,
			eventMessages,
			eventData,
		)

		if err != nil {
			return err
		}

		return commit(ctx)
	})

	if err != nil {
		return nil, err
	}

	return res, nil
}

func bulkStepRunEvents(
	ctx context.Context,
	l *zerolog.Logger,
	dbtx dbsqlc.DBTX,
	queries *dbsqlc.Queries,
	stepRunIds []pgtype.UUID,
	timeSeen []pgtype.Timestamp,
	reasons []dbsqlc.StepRunEventReason,
	severities []dbsqlc.StepRunEventSeverity,
	messages []string,
	data []map[string]interface{},
) error {
	inputData := [][]byte{}
	inputReasons := []string{}
	inputSeverities := []string{}

	for _, d := range data {
		dataBytes, err := json.Marshal(d)

		if err != nil {
			l.Err(err).Msg("could not marshal deferred step run event data")
			return err
		}

		inputData = append(inputData, dataBytes)
	}

	for _, r := range reasons {
		inputReasons = append(inputReasons, string(r))
	}

	for _, s := range severities {
		inputSeverities = append(inputSeverities, string(s))
	}

	err := queries.BulkCreateStepRunEvent(ctx, dbtx, dbsqlc.BulkCreateStepRunEventParams{
		Steprunids: stepRunIds,
		Reasons:    inputReasons,
		Severities: inputSeverities,
		Messages:   messages,
		Data:       inputData,
		Timeseen:   timeSeen,
	})

	if err != nil {
		l.Error().Err(err).Msg("could not create deferred step run event")
		return fmt.Errorf("bulk_events - could not create deferred step run event: %w", err)
	}

	return nil
}
