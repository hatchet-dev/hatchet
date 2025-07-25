package olap

import (
	"context"
	"errors"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	msgqueue "github.com/hatchet-dev/hatchet/internal/msgqueue/v1"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	"github.com/hatchet-dev/hatchet/pkg/repository/postgres/sqlchelpers"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository/v1"
	"github.com/hatchet-dev/hatchet/pkg/repository/v1/sqlcv1"
)

type OLAPController interface {
	Start(ctx context.Context) error
}

type OLAPControllerImpl struct {
	mq                           msgqueue.MessageQueue
	l                            *zerolog.Logger
	repo                         v1.Repository
	dv                           datautils.DataDecoderValidator
	a                            *hatcheterrors.Wrapped
	p                            *partition.Partition
	s                            gocron.Scheduler
	ta                           *alerting.TenantAlertManager
	processTenantAlertOperations *queueutils.OperationPool
	samplingHashThreshold        *int64
	olapConfig                   *server.ConfigFileOperations
	prometheusMetricsEnabled     bool
}

type OLAPControllerOpt func(*OLAPControllerOpts)

type OLAPControllerOpts struct {
	mq                       msgqueue.MessageQueue
	l                        *zerolog.Logger
	repo                     v1.Repository
	dv                       datautils.DataDecoderValidator
	alerter                  hatcheterrors.Alerter
	p                        *partition.Partition
	ta                       *alerting.TenantAlertManager
	samplingHashThreshold    *int64
	olapConfig               *server.ConfigFileOperations
	prometheusMetricsEnabled bool
}

func defaultOLAPControllerOpts() *OLAPControllerOpts {
	l := logger.NewDefaultLogger("olap-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &OLAPControllerOpts{
		l:                        &l,
		dv:                       datautils.NewDataDecoderValidator(),
		alerter:                  alerter,
		prometheusMetricsEnabled: false,
	}
}

func WithMessageQueue(mq msgqueue.MessageQueue) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.mq = mq
	}
}

func WithLogger(l *zerolog.Logger) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.l = l
	}
}

func WithAlerter(a hatcheterrors.Alerter) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.alerter = a
	}
}

func WithRepository(r v1.Repository) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.repo = r
	}
}

func WithPartition(p *partition.Partition) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.p = p
	}
}

func WithDataDecoderValidator(dv datautils.DataDecoderValidator) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.dv = dv
	}
}

func WithTenantAlertManager(ta *alerting.TenantAlertManager) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.ta = ta
	}
}

func WithSamplingConfig(c server.ConfigFileSampling) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		if c.Enabled && c.SamplingRate != 1.0 {
			// convert the rate into a hash threshold
			hashThreshold := int64(c.SamplingRate * 100)

			opts.samplingHashThreshold = &hashThreshold
		}
	}
}

func WithOperationsConfig(c server.ConfigFileOperations) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.olapConfig = &c
	}
}

func WithPrometheusMetricsEnabled(enabled bool) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.prometheusMetricsEnabled = enabled
	}
}

func New(fs ...OLAPControllerOpt) (*OLAPControllerImpl, error) {
	opts := defaultOLAPControllerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.mq == nil {
		return nil, fmt.Errorf("task queue is required. use WithMessageQueue")
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.p == nil {
		return nil, errors.New("partition is required. use WithPartition")
	}

	if opts.ta == nil {
		return nil, errors.New("tenant alerter is required. use WithTenantAlertManager")
	}

	newLogger := opts.l.With().Str("service", "olap-controller").Logger()
	opts.l = &newLogger

	s, err := gocron.NewScheduler(gocron.WithLocation(time.UTC))

	if err != nil {
		return nil, fmt.Errorf("could not create scheduler: %w", err)
	}

	a := hatcheterrors.NewWrapped(opts.alerter)
	a.WithData(map[string]interface{}{"service": "olap-controller"})

	o := &OLAPControllerImpl{
		mq:                       opts.mq,
		l:                        opts.l,
		s:                        s,
		p:                        opts.p,
		repo:                     opts.repo,
		dv:                       opts.dv,
		a:                        a,
		ta:                       opts.ta,
		samplingHashThreshold:    opts.samplingHashThreshold,
		olapConfig:               opts.olapConfig,
		prometheusMetricsEnabled: opts.prometheusMetricsEnabled,
	}

	// Default jitter value
	jitter := 1500 * time.Millisecond

	// Override with config value if available
	if o.olapConfig != nil && o.olapConfig.Jitter > 0 {
		jitter = time.Duration(o.olapConfig.Jitter) * time.Millisecond
	}

	// Default timeout
	timeout := 15 * time.Second

	o.processTenantAlertOperations = queueutils.NewOperationPool(
		opts.l,
		timeout,
		"process tenant alerts",
		o.processTenantAlerts,
	).WithJitter(jitter)

	return o, nil
}

func (o *OLAPControllerImpl) Start() (func() error, error) {
	cleanupHeavyReadMQ, heavyReadMQ := o.mq.Clone()
	heavyReadMQ.SetQOS(2000)

	o.s.Start()

	mqBuffer := msgqueue.NewMQSubBuffer(msgqueue.OLAP_QUEUE, heavyReadMQ, o.handleBufferedMsgs)
	wg := sync.WaitGroup{}

	startupPartitionCtx, cancelStartupPartition := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelStartupPartition()

	// always create table partition on startup
	if err := o.createTablePartition(startupPartitionCtx); err != nil {
		return nil, fmt.Errorf("could not create table partition: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	_, err := o.s.NewJob(
		gocron.DurationJob(time.Minute*15),
		gocron.NewTask(
			o.runOLAPTablePartition(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule task table partition: %w", err)
	}

	// Default poll interval
	pollIntervalSec := 2

	// Override with config value if available
	if o.olapConfig != nil && o.olapConfig.PollInterval > 0 {
		pollIntervalSec = o.olapConfig.PollInterval
	}

	_, err = o.s.NewJob(
		gocron.DurationJob(time.Second*time.Duration(pollIntervalSec)),
		gocron.NewTask(
			o.runTaskStatusUpdates(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule task status updates: %w", err)
	}

	_, err = o.s.NewJob(
		gocron.DurationJob(time.Second*time.Duration(pollIntervalSec)),
		gocron.NewTask(
			o.runDAGStatusUpdates(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule dag status updates: %w", err)
	}

	_, err = o.s.NewJob(
		gocron.DurationJob(time.Second*60),
		gocron.NewTask(
			o.runTenantProcessAlerts(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not schedule process tenant alerts: %w", err)
	}

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		cancel()

		if err := cleanupBuffer(); err != nil {
			return err
		}

		if err := cleanupHeavyReadMQ(); err != nil {
			return err
		}

		if err := o.s.Shutdown(); err != nil {
			return err
		}

		wg.Wait()

		return nil
	}

	return cleanup, nil
}

func (tc *OLAPControllerImpl) handleBufferedMsgs(tenantId, msgId string, payloads [][]byte) (err error) {
	defer func() {
		if r := recover(); r != nil {
			recoverErr := recoveryutils.RecoverWithAlert(tc.l, tc.a, r)

			if recoverErr != nil {
				err = recoverErr
			}
		}
	}()

	switch msgId {
	case "created-task":
		return tc.handleCreatedTask(context.Background(), tenantId, payloads)
	case "created-dag":
		return tc.handleCreatedDAG(context.Background(), tenantId, payloads)
	case "create-monitoring-event":
		return tc.handleCreateMonitoringEvent(context.Background(), tenantId, payloads)
	case "created-event-trigger":
		return tc.handleCreateEventTriggers(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedTask(ctx context.Context, tenantId string, payloads [][]byte) error {
	createTaskOpts := make([]*sqlcv1.V1Task, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

	for _, msg := range msgs {
		if !tc.sample(sqlchelpers.UUIDToStr(msg.WorkflowRunID)) {
			tc.l.Debug().Msgf("skipping task %d for workflow run %s", msg.ID, sqlchelpers.UUIDToStr(msg.WorkflowRunID))
			continue
		}

		createTaskOpts = append(createTaskOpts, msg.V1Task)
	}

	return tc.repo.OLAP().CreateTasks(ctx, tenantId, createTaskOpts)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedDAG(ctx context.Context, tenantId string, payloads [][]byte) error {
	createDAGOpts := make([]*v1.DAGWithData, 0)
	msgs := msgqueue.JSONConvert[tasktypes.CreatedDAGPayload](payloads)

	for _, msg := range msgs {
		if !tc.sample(sqlchelpers.UUIDToStr(msg.ExternalID)) {
			tc.l.Debug().Msgf("skipping dag %s", sqlchelpers.UUIDToStr(msg.ExternalID))
			continue
		}

		createDAGOpts = append(createDAGOpts, msg.DAGWithData)
	}

	return tc.repo.OLAP().CreateDAGs(ctx, tenantId, createDAGOpts)
}

func (tc *OLAPControllerImpl) handleCreateEventTriggers(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.CreatedEventTriggerPayload](payloads)

	seenEventKeysSet := make(map[string]bool)

	bulkCreateTriggersParams := make([]v1.EventTriggersFromExternalId, 0)

	tenantIds := make([]pgtype.UUID, 0)
	externalIds := make([]pgtype.UUID, 0)
	seenAts := make([]pgtype.Timestamptz, 0)
	keys := make([]string, 0)
	payloadstoInsert := make([][]byte, 0)
	additionalMetadatas := make([][]byte, 0)
	scopes := make([]*string, 0)

	for _, msg := range msgs {
		for _, payload := range msg.Payloads {
			if payload.MaybeRunId != nil && payload.MaybeRunInsertedAt != nil {
				var filterId pgtype.UUID

				if payload.FilterId != nil {
					filterId = sqlchelpers.UUIDFromStr(*payload.FilterId)
				}

				bulkCreateTriggersParams = append(bulkCreateTriggersParams, v1.EventTriggersFromExternalId{
					RunID:           *payload.MaybeRunId,
					RunInsertedAt:   sqlchelpers.TimestamptzFromTime(*payload.MaybeRunInsertedAt),
					EventExternalId: sqlchelpers.UUIDFromStr(payload.EventExternalId),
					EventSeenAt:     sqlchelpers.TimestamptzFromTime(payload.EventSeenAt),
					FilterId:        filterId,
				})
			}

			_, eventAlreadySeen := seenEventKeysSet[payload.EventExternalId]

			if eventAlreadySeen {
				continue
			}

			seenEventKeysSet[payload.EventExternalId] = true
			tenantIds = append(tenantIds, sqlchelpers.UUIDFromStr(tenantId))
			externalIds = append(externalIds, sqlchelpers.UUIDFromStr(payload.EventExternalId))
			seenAts = append(seenAts, sqlchelpers.TimestamptzFromTime(payload.EventSeenAt))
			keys = append(keys, payload.EventKey)
			payloadstoInsert = append(payloadstoInsert, payload.EventPayload)
			additionalMetadatas = append(additionalMetadatas, payload.EventAdditionalMetadata)
			scopes = append(scopes, payload.EventScope)
		}
	}

	bulkCreateEventParams := sqlcv1.BulkCreateEventsParams{
		Tenantids:           tenantIds,
		Externalids:         externalIds,
		Seenats:             seenAts,
		Keys:                keys,
		Payloads:            payloadstoInsert,
		Additionalmetadatas: additionalMetadatas,
		Scopes:              scopes,
	}

	return tc.repo.OLAP().BulkCreateEventsAndTriggers(
		ctx,
		bulkCreateEventParams,
		bulkCreateTriggersParams,
	)
}

// handleCreateMonitoringEvent is responsible for sending a group of monitoring events to the OLAP repository
func (tc *OLAPControllerImpl) handleCreateMonitoringEvent(ctx context.Context, tenantId string, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.CreateMonitoringEventPayload](payloads)

	taskIdsToLookup := make([]int64, len(msgs))

	for i, msg := range msgs {
		taskIdsToLookup[i] = msg.TaskId
	}

	metas, err := tc.repo.Tasks().ListTaskMetas(ctx, tenantId, taskIdsToLookup)

	if err != nil {
		return err
	}

	taskIdsToMetas := make(map[int64]*sqlcv1.ListTaskMetasRow)

	for _, taskMeta := range metas {
		taskIdsToMetas[taskMeta.ID] = taskMeta
	}

	taskIds := make([]int64, 0)
	taskInsertedAts := make([]pgtype.Timestamptz, 0)
	retryCounts := make([]int32, 0)
	workerIds := make([]string, 0)
	workflowIds := make([]pgtype.UUID, 0)
	eventTypes := make([]sqlcv1.V1EventTypeOlap, 0)
	readableStatuses := make([]sqlcv1.V1ReadableStatusOlap, 0)
	eventPayloads := make([]string, 0)
	eventMessages := make([]string, 0)
	timestamps := make([]pgtype.Timestamptz, 0)

	for _, msg := range msgs {
		taskMeta := taskIdsToMetas[msg.TaskId]

		if taskMeta == nil {
			tc.l.Error().Msgf("could not find task meta for task id %d", msg.TaskId)
			continue
		}

		if !tc.sample(sqlchelpers.UUIDToStr(taskMeta.WorkflowRunID)) {
			tc.l.Debug().Msgf("skipping task %d for workflow run %s", msg.TaskId, sqlchelpers.UUIDToStr(taskMeta.WorkflowRunID))
			continue
		}

		taskIds = append(taskIds, msg.TaskId)
		taskInsertedAts = append(taskInsertedAts, taskMeta.InsertedAt)
		workflowIds = append(workflowIds, taskMeta.WorkflowID)
		retryCounts = append(retryCounts, msg.RetryCount)
		eventTypes = append(eventTypes, msg.EventType)
		eventPayloads = append(eventPayloads, msg.EventPayload)
		eventMessages = append(eventMessages, msg.EventMessage)
		timestamps = append(timestamps, sqlchelpers.TimestamptzFromTime(msg.EventTimestamp))

		if msg.WorkerId != nil {
			workerIds = append(workerIds, *msg.WorkerId)
		} else {
			workerIds = append(workerIds, "")
		}

		switch msg.EventType {
		case sqlcv1.V1EventTypeOlapRETRYING:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapREASSIGNED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapRETRIEDBYUSER:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapCREATED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapQUEUED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapREQUEUEDNOWORKER:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapREQUEUEDRATELIMIT:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapQUEUED)
		case sqlcv1.V1EventTypeOlapASSIGNED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapACKNOWLEDGED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapSENTTOWORKER:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapSLOTRELEASED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapSTARTED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapTIMEOUTREFRESHED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapSCHEDULINGTIMEDOUT:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapFINISHED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCOMPLETED)
		case sqlcv1.V1EventTypeOlapFAILED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapCANCELLED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCANCELLED)
		case sqlcv1.V1EventTypeOlapTIMEDOUT:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapRATELIMITERROR:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapSKIPPED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCOMPLETED)
		}
	}

	opts := make([]sqlcv1.CreateTaskEventsOLAPParams, 0)

	for i, taskId := range taskIds {
		var workerId pgtype.UUID

		if workerIds[i] != "" {
			workerId = sqlchelpers.UUIDFromStr(workerIds[i])
		}

		event := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:               sqlchelpers.UUIDFromStr(tenantId),
			TaskID:                 taskId,
			TaskInsertedAt:         taskInsertedAts[i],
			WorkflowID:             workflowIds[i],
			EventType:              eventTypes[i],
			EventTimestamp:         timestamps[i],
			ReadableStatus:         readableStatuses[i],
			RetryCount:             retryCounts[i],
			WorkerID:               workerId,
			AdditionalEventMessage: sqlchelpers.TextFromStr(eventMessages[i]),
		}

		switch eventTypes[i] {
		case sqlcv1.V1EventTypeOlapFINISHED:
			if eventPayloads[i] != "" {
				event.Output = []byte(eventPayloads[i])
			}
		case sqlcv1.V1EventTypeOlapFAILED:
			event.ErrorMessage = sqlchelpers.TextFromStr(eventPayloads[i])
		case sqlcv1.V1EventTypeOlapCANCELLED:
			event.AdditionalEventMessage = sqlchelpers.TextFromStr(eventMessages[i])
		}

		opts = append(opts, event)
	}

	return tc.repo.OLAP().CreateTaskEvents(ctx, tenantId, opts)
}

func (tc *OLAPControllerImpl) sample(workflowRunID string) bool {
	if tc.samplingHashThreshold == nil {
		return true
	}

	bucket := hashToBucket(workflowRunID, 100)

	return int64(bucket) < *tc.samplingHashThreshold
}

func hashToBucket(workflowRunID string, buckets int) int {
	hasher := fnv.New32a()
	idBytes := []byte(workflowRunID)
	hasher.Write(idBytes)
	return int(hasher.Sum32()) % buckets
}
