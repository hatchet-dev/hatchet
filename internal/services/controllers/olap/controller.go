package olap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"math/rand"
	"sync"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/codes"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/integrations/alerting"
	"github.com/hatchet-dev/hatchet/internal/msgqueue"
	"github.com/hatchet-dev/hatchet/internal/queueutils"
	"github.com/hatchet-dev/hatchet/internal/services/partition"
	"github.com/hatchet-dev/hatchet/internal/services/shared/recoveryutils"
	tasktypes "github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes/v1"
	"github.com/hatchet-dev/hatchet/pkg/config/server"
	hatcheterrors "github.com/hatchet-dev/hatchet/pkg/errors"
	"github.com/hatchet-dev/hatchet/pkg/logger"
	v1 "github.com/hatchet-dev/hatchet/pkg/repository"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlchelpers"
	"github.com/hatchet-dev/hatchet/pkg/repository/sqlcv1"
	"github.com/hatchet-dev/hatchet/pkg/telemetry"
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
	analyzeCronInterval          time.Duration
	taskPrometheusUpdateCh       chan taskPrometheusUpdate
	taskPrometheusWorkerCtx      context.Context
	taskPrometheusWorkerCancel   context.CancelFunc
	dagPrometheusUpdateCh        chan dagPrometheusUpdate
	dagPrometheusWorkerCtx       context.Context
	dagPrometheusWorkerCancel    context.CancelFunc
	statusUpdateBatchSizeLimits  v1.StatusUpdateBatchSizeLimits
}

type OLAPControllerOpt func(*OLAPControllerOpts)

type OLAPControllerOpts struct {
	mq                          msgqueue.MessageQueue
	l                           *zerolog.Logger
	repo                        v1.Repository
	dv                          datautils.DataDecoderValidator
	alerter                     hatcheterrors.Alerter
	p                           *partition.Partition
	ta                          *alerting.TenantAlertManager
	samplingHashThreshold       *int64
	olapConfig                  *server.ConfigFileOperations
	prometheusMetricsEnabled    bool
	analyzeCronInterval         time.Duration
	statusUpdateBatchSizeLimits v1.StatusUpdateBatchSizeLimits
}

func defaultOLAPControllerOpts() *OLAPControllerOpts {
	l := logger.NewDefaultLogger("olap-controller")
	alerter := hatcheterrors.NoOpAlerter{}

	return &OLAPControllerOpts{
		l:                        &l,
		dv:                       datautils.NewDataDecoderValidator(),
		alerter:                  alerter,
		prometheusMetricsEnabled: false,
		analyzeCronInterval:      3 * time.Hour,
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

func WithAnalyzeCronInterval(interval time.Duration) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.analyzeCronInterval = interval
	}
}

func WithOLAPStatusUpdateBatchSizeLimits(limits v1.StatusUpdateBatchSizeLimits) OLAPControllerOpt {
	return func(opts *OLAPControllerOpts) {
		opts.statusUpdateBatchSizeLimits = limits
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

	// Channel size = 2 * batch_size * num_partitions for overhead
	// batch_size = 1000, num_partitions from partition config
	numPartitions := 4
	prometheusChannelSize := 2 * 1000 * numPartitions

	taskPrometheusUpdateCh := make(chan taskPrometheusUpdate, prometheusChannelSize)
	dagPrometheusUpdateCh := make(chan dagPrometheusUpdate, prometheusChannelSize)

	o := &OLAPControllerImpl{
		mq:                          opts.mq,
		l:                           opts.l,
		s:                           s,
		p:                           opts.p,
		repo:                        opts.repo,
		dv:                          opts.dv,
		a:                           a,
		ta:                          opts.ta,
		samplingHashThreshold:       opts.samplingHashThreshold,
		olapConfig:                  opts.olapConfig,
		prometheusMetricsEnabled:    opts.prometheusMetricsEnabled,
		analyzeCronInterval:         opts.analyzeCronInterval,
		taskPrometheusUpdateCh:      taskPrometheusUpdateCh,
		dagPrometheusUpdateCh:       dagPrometheusUpdateCh,
		statusUpdateBatchSizeLimits: opts.statusUpdateBatchSizeLimits,
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
	cleanupHeavyReadMQ, heavyReadMQ, err := o.mq.Clone()

	if err != nil {
		return nil, err
	}
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

	// Start prometheus workers if metrics are enabled
	if o.prometheusMetricsEnabled {
		o.taskPrometheusWorkerCtx, o.taskPrometheusWorkerCancel = context.WithCancel(context.Background())
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.runTaskPrometheusUpdateWorker()
		}()

		o.dagPrometheusWorkerCtx, o.dagPrometheusWorkerCancel = context.WithCancel(context.Background())
		wg.Add(1)
		go func() {
			defer wg.Done()
			o.runDAGPrometheusUpdateWorker()
		}()
	}

	_, err = o.s.NewJob(
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

	_, err = o.s.NewJob(
		gocron.DurationJob(o.analyzeCronInterval),
		gocron.NewTask(
			o.runAnalyze(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not run analyze: %w", err)
	}

	_, err = o.s.NewJob(
		gocron.DurationJob(o.repo.Payloads().ExternalCutoverProcessInterval()),
		gocron.NewTask(
			o.processPayloadExternalCutovers(ctx),
		),
		gocron.WithSingletonMode(gocron.LimitModeReschedule),
	)

	if err != nil {
		wrappedErr := fmt.Errorf("could not schedule process olap payload external cutovers: %w", err)

		cancel()

		return nil, wrappedErr
	}

	cleanupBuffer, err := mqBuffer.Start()

	if err != nil {
		cancel()
		return nil, fmt.Errorf("could not start message queue buffer: %w", err)
	}

	cleanup := func() error {
		cancel()

		// Stop prometheus workers if running
		if o.taskPrometheusWorkerCancel != nil {
			o.taskPrometheusWorkerCancel()
		}
		if o.dagPrometheusWorkerCancel != nil {
			o.dagPrometheusWorkerCancel()
		}

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

		// Close prometheus channels after all workers are done
		if o.taskPrometheusUpdateCh != nil {
			close(o.taskPrometheusUpdateCh)
		}
		if o.dagPrometheusUpdateCh != nil {
			close(o.dagPrometheusUpdateCh)
		}

		return nil
	}

	return cleanup, nil
}

func (tc *OLAPControllerImpl) handleBufferedMsgs(tenantId uuid.UUID, msgId string, payloads [][]byte) (err error) {
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
	case "failed-webhook-validation":
		return tc.handleFailedWebhookValidation(context.Background(), tenantId, payloads)
	case "cel-evaluation-failure":
		return tc.handleCelEvaluationFailure(context.Background(), tenantId, payloads)
	case "offload-payload":
		return tc.handlePayloadOffload(context.Background(), tenantId, payloads)
	}

	return fmt.Errorf("unknown message id: %s", msgId)
}

func (tc *OLAPControllerImpl) handlePayloadOffload(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	offloads := make([]v1.OffloadPayloadOpts, 0)

	msgs := msgqueue.JSONConvert[v1.OLAPPayloadsToOffload](payloads)

	for _, msg := range msgs {
		for _, payload := range msg.Payloads {
			if !tc.sample(payload.ExternalId.String()) {
				tc.l.Debug().Msgf("skipping payload offload external id %s", payload.ExternalId)
				continue
			}

			offloads = append(offloads, v1.OffloadPayloadOpts(payload))
		}
	}

	return tc.repo.OLAP().OffloadPayloads(ctx, tenantId, offloads)
}

func (tc *OLAPControllerImpl) handleCelEvaluationFailure(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	failures := make([]v1.CELEvaluationFailure, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CELEvaluationFailures](payloads)

	for _, msg := range msgs {
		for _, failure := range msg.Failures {
			if !tc.sample(failure.ErrorMessage) {
				tc.l.Debug().Msgf("skipping CEL evaluation failure %s for source %s", failure.ErrorMessage, failure.Source)
				continue
			}

			failures = append(failures, failure)
		}
	}

	return tc.repo.OLAP().StoreCELEvaluationFailures(ctx, tenantId, failures)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedTask(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	createTaskOpts := make([]*v1.V1TaskWithPayload, 0)

	msgs := msgqueue.JSONConvert[tasktypes.CreatedTaskPayload](payloads)

	for _, msg := range msgs {
		if !tc.sample(msg.WorkflowRunID.String()) {
			tc.l.Debug().Msgf("skipping task %d for workflow run %s", msg.ID, msg.WorkflowRunID.String())
			continue
		}

		createTaskOpts = append(createTaskOpts, msg.V1TaskWithPayload)
	}

	return tc.repo.OLAP().CreateTasks(ctx, tenantId, createTaskOpts)
}

// handleCreatedTask is responsible for flushing a created task to the OLAP repository
func (tc *OLAPControllerImpl) handleCreatedDAG(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	createDAGOpts := make([]*v1.DAGWithData, 0)
	msgs := msgqueue.JSONConvert[tasktypes.CreatedDAGPayload](payloads)

	for _, msg := range msgs {
		if !tc.sample(msg.ExternalID.String()) {
			tc.l.Debug().Msgf("skipping dag %s", msg.ExternalID.String())
			continue
		}

		createDAGOpts = append(createDAGOpts, msg.DAGWithData)
	}

	return tc.repo.OLAP().CreateDAGs(ctx, tenantId, createDAGOpts)
}

func (tc *OLAPControllerImpl) handleCreateEventTriggers(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	msgs := msgqueue.JSONConvert[tasktypes.CreatedEventTriggerPayload](payloads)

	seenEventKeysSet := make(map[uuid.UUID]bool)

	bulkCreateTriggersParams := make([]v1.EventTriggersFromExternalId, 0)

	tenantIds := make([]uuid.UUID, 0)
	externalIds := make([]uuid.UUID, 0)
	seenAts := make([]pgtype.Timestamptz, 0)
	keys := make([]string, 0)
	payloadstoInsert := make([][]byte, 0)
	additionalMetadatas := make([][]byte, 0)
	scopes := make([]pgtype.Text, 0)
	triggeringWebhookNames := make([]pgtype.Text, 0)

	for _, msg := range msgs {
		for _, payload := range msg.Payloads {
			if payload.MaybeRunId != nil && payload.MaybeRunInsertedAt != nil {
				var filterId uuid.UUID

				if payload.FilterId != nil {
					filterId = *payload.FilterId
				}

				bulkCreateTriggersParams = append(bulkCreateTriggersParams, v1.EventTriggersFromExternalId{
					RunID:           *payload.MaybeRunId,
					RunInsertedAt:   sqlchelpers.TimestamptzFromTime(*payload.MaybeRunInsertedAt),
					EventExternalId: payload.EventExternalId,
					EventSeenAt:     sqlchelpers.TimestamptzFromTime(payload.EventSeenAt),
					FilterId:        filterId,
				})
			}

			_, eventAlreadySeen := seenEventKeysSet[payload.EventExternalId]

			if eventAlreadySeen {
				continue
			}

			seenEventKeysSet[payload.EventExternalId] = true
			tenantIds = append(tenantIds, tenantId)
			externalIds = append(externalIds, payload.EventExternalId)
			seenAts = append(seenAts, sqlchelpers.TimestamptzFromTime(payload.EventSeenAt))
			keys = append(keys, payload.EventKey)
			payloadstoInsert = append(payloadstoInsert, payload.EventPayload)
			additionalMetadatas = append(additionalMetadatas, payload.EventAdditionalMetadata)

			var scope pgtype.Text
			if payload.EventScope != nil {
				scope = sqlchelpers.TextFromStr(*payload.EventScope)
			}

			scopes = append(scopes, scope)

			var triggeringWebhookName pgtype.Text
			if payload.TriggeringWebhookName != nil {
				triggeringWebhookName = sqlchelpers.TextFromStr(*payload.TriggeringWebhookName)
			}

			triggeringWebhookNames = append(triggeringWebhookNames, triggeringWebhookName)
		}
	}

	bulkCreateEventParams := sqlcv1.BulkCreateEventsOLAPParams{
		Tenantids:              tenantIds,
		Externalids:            externalIds,
		Seenats:                seenAts,
		Keys:                   keys,
		Payloads:               payloadstoInsert,
		Additionalmetadatas:    additionalMetadatas,
		Scopes:                 scopes,
		TriggeringWebhookNames: triggeringWebhookNames,
	}

	return tc.repo.OLAP().BulkCreateEventsAndTriggers(
		ctx,
		bulkCreateEventParams,
		bulkCreateTriggersParams,
	)
}

// handleCreateMonitoringEvent is responsible for sending a group of monitoring events to the OLAP repository
func (tc *OLAPControllerImpl) handleCreateMonitoringEvent(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
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
	workerIds := make([]uuid.UUID, 0)
	workflowIds := make([]uuid.UUID, 0)
	eventTypes := make([]sqlcv1.V1EventTypeOlap, 0)
	readableStatuses := make([]sqlcv1.V1ReadableStatusOlap, 0)
	eventPayloads := make([]string, 0)
	eventMessages := make([]string, 0)
	timestamps := make([]pgtype.Timestamptz, 0)
	eventExternalIds := make([]*uuid.UUID, 0)

	for _, msg := range msgs {
		taskMeta := taskIdsToMetas[msg.TaskId]

		if taskMeta == nil {
			tc.l.Error().Msgf("could not find task meta for task id %d", msg.TaskId)
			continue
		}

		if !tc.sample(taskMeta.WorkflowRunID.String()) {
			tc.l.Debug().Msgf("skipping task %d for workflow run %s", msg.TaskId, taskMeta.WorkflowRunID.String())
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
		externalId := uuid.New()
		eventExternalIds = append(eventExternalIds, &externalId)

		if msg.WorkerId != nil {
			workerIds = append(workerIds, *msg.WorkerId)
		} else {
			workerIds = append(workerIds, uuid.Nil)
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
			// Backwards compatibility (older clients).
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCANCELLED)
		case sqlcv1.V1EventTypeOlapCANCELLING:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCANCELLED)
		case sqlcv1.V1EventTypeOlapCANCELLEDCONFIRMED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCANCELLED)
		case sqlcv1.V1EventTypeOlapCANCELLATIONFAILED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCANCELLED)
		case sqlcv1.V1EventTypeOlapDURABLEEVICTED:
			// TODO-DURABLE: i'm not sure what we want to do here for status...
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapDURABLERESUMING:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapRUNNING)
		case sqlcv1.V1EventTypeOlapTIMEDOUT:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapRATELIMITERROR:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		case sqlcv1.V1EventTypeOlapSKIPPED:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapCOMPLETED)
		case sqlcv1.V1EventTypeOlapCOULDNOTSENDTOWORKER:
			readableStatuses = append(readableStatuses, sqlcv1.V1ReadableStatusOlapFAILED)
		}
	}

	opts := make([]sqlcv1.CreateTaskEventsOLAPParams, 0)

	for i, taskId := range taskIds {
		var workerId *uuid.UUID

		if workerIds[i] != uuid.Nil {
			workerId = &workerIds[i]
		}

		event := sqlcv1.CreateTaskEventsOLAPParams{
			TenantID:               tenantId,
			TaskID:                 taskId,
			TaskInsertedAt:         taskInsertedAts[i],
			WorkflowID:             workflowIds[i],
			EventType:              eventTypes[i],
			EventTimestamp:         timestamps[i],
			ReadableStatus:         readableStatuses[i],
			RetryCount:             retryCounts[i],
			WorkerID:               workerId,
			AdditionalEventMessage: sqlchelpers.TextFromStr(eventMessages[i]),
			ExternalID:             eventExternalIds[i],
		}

		// For worker-emitted monitoring events, we often only get structured JSON in `event_payload`.
		// The public API exposes the human message (`additional__event_message`) but not the data field,
		// so derive a message from the payload when none was provided.
		if eventMessages[i] == "" && eventPayloads[i] != "" {
			switch eventTypes[i] {
			case sqlcv1.V1EventTypeOlapCANCELLING, sqlcv1.V1EventTypeOlapCANCELLEDCONFIRMED, sqlcv1.V1EventTypeOlapCANCELLATIONFAILED:
				var payload struct {
					Reason    string `json:"reason"`
					ElapsedMs *int   `json:"elapsed_ms"`
				}
				if err := json.Unmarshal([]byte(eventPayloads[i]), &payload); err == nil {
					if payload.Reason != "" || payload.ElapsedMs != nil {
						msg := fmt.Sprintf("reason=%s", payload.Reason)
						if payload.Reason == "" {
							msg = "reason=unknown"
						}
						if payload.ElapsedMs != nil {
							msg = fmt.Sprintf("%s elapsed_ms=%d", msg, *payload.ElapsedMs)
						}
						event.AdditionalEventMessage = sqlchelpers.TextFromStr(msg)
					}
				}
			case sqlcv1.V1EventTypeOlapDURABLEEVICTED, sqlcv1.V1EventTypeOlapDURABLERESUMING:
				var payload struct {
					Reason     string  `json:"reason"`
					WaitKind   *string `json:"wait_kind"`
					ResourceId *string `json:"resource_id"`
				}
				if err := json.Unmarshal([]byte(eventPayloads[i]), &payload); err == nil {
					msg := payload.Reason
					if msg == "" {
						msg = "durable"
					}
					if payload.WaitKind != nil && *payload.WaitKind != "" {
						msg = fmt.Sprintf("%s wait_kind=%s", msg, *payload.WaitKind)
					}
					if payload.ResourceId != nil && *payload.ResourceId != "" {
						msg = fmt.Sprintf("%s resource_id=%s", msg, *payload.ResourceId)
					}
					event.AdditionalEventMessage = sqlchelpers.TextFromStr(msg)
				}
			}
		}

		switch eventTypes[i] {
		case sqlcv1.V1EventTypeOlapFINISHED:
			if eventPayloads[i] != "" {
				event.Output = []byte(eventPayloads[i])
			}
		case sqlcv1.V1EventTypeOlapFAILED:
			event.ErrorMessage = sqlchelpers.TextFromStr(eventPayloads[i])
		case sqlcv1.V1EventTypeOlapCANCELLED,
			sqlcv1.V1EventTypeOlapCANCELLING,
			sqlcv1.V1EventTypeOlapCANCELLEDCONFIRMED,
			sqlcv1.V1EventTypeOlapCANCELLATIONFAILED,
			sqlcv1.V1EventTypeOlapDURABLEEVICTED,
			sqlcv1.V1EventTypeOlapDURABLERESUMING:
			// Keep message in `additional__event_message`, and store structured details in `additional__event_data`.
			if eventPayloads[i] != "" {
				event.AdditionalEventData = sqlchelpers.TextFromStr(eventPayloads[i])
			}
		}

		opts = append(opts, event)
	}

	err = tc.repo.OLAP().CreateTaskEvents(ctx, tenantId, opts)

	if err != nil {
		return err
	}

	if !tc.repo.OLAP().PayloadStore().ExternalStoreEnabled() {
		return nil
	}

	offloadToExternalOpts := make([]v1.OffloadToExternalStoreOpts, 0)
	idInsertedAtToExternalId := make(map[v1.IdInsertedAt]*uuid.UUID)

	for _, opt := range opts {
		// generating a dummy id + inserted at to use for creating the external keys for the task events
		// we do this since we don't have the id + inserted at of the events themselves on the opts, and we don't
		// actually need those for anything once the keys are created.
		dummyId := rand.Int63()
		// randomly jitter the inserted at time by +/- 300ms to make collisions virtually impossible
		dummyInsertedAt := time.Now().Add(time.Duration(rand.Intn(2*300+1)-300) * time.Millisecond)

		idInsertedAtToExternalId[v1.IdInsertedAt{
			ID:         dummyId,
			InsertedAt: sqlchelpers.TimestamptzFromTime(dummyInsertedAt),
		}] = opt.ExternalID

		offloadToExternalOpts = append(offloadToExternalOpts, v1.OffloadToExternalStoreOpts{
			TenantId:   tenantId,
			ExternalID: *opt.ExternalID,
			InsertedAt: sqlchelpers.TimestamptzFromTime(dummyInsertedAt),
			Payload:    opt.Output,
		})
	}

	if len(offloadToExternalOpts) == 0 {
		return nil
	}

	// retrieveOptsToKey, err := tc.repo.OLAP().PayloadStore().ExternalStore().Store(ctx, offloadToExternalOpts...)

	// if err != nil {
	// 	return err
	// }

	// offloadOpts := make([]v1.OffloadPayloadOpts, 0)

	// for opt, key := range retrieveOptsToKey {
	// 	externalId := idInsertedAtToExternalId[v1.IdInsertedAt{
	// 		ID:         opt.Id,
	// 		InsertedAt: opt.InsertedAt,
	// 	}]

	// 	offloadOpts = append(offloadOpts, v1.OffloadPayloadOpts{
	// 		ExternalId:          externalId,
	// 		ExternalLocationKey: string(key),
	// 	})
	// }

	// err = tc.repo.OLAP().OffloadPayloads(ctx, tenantId, offloadOpts)

	// if err != nil {
	// 	return err
	// }

	return nil
}

func (tc *OLAPControllerImpl) handleFailedWebhookValidation(ctx context.Context, tenantId uuid.UUID, payloads [][]byte) error {
	createFailedWebhookValidationOpts := make([]v1.CreateIncomingWebhookFailureLogOpts, 0)

	msgs := msgqueue.JSONConvert[tasktypes.FailedWebhookValidationPayload](payloads)

	for _, msg := range msgs {
		if !tc.sample(msg.ErrorText) {
			tc.l.Debug().Msgf("skipping failure logging for webhook %s", msg.WebhookName)
			continue
		}

		createFailedWebhookValidationOpts = append(createFailedWebhookValidationOpts, v1.CreateIncomingWebhookFailureLogOpts{
			WebhookName: msg.WebhookName,
			ErrorText:   msg.ErrorText,
		})
	}

	return tc.repo.OLAP().CreateIncomingWebhookValidationFailureLogs(ctx, tenantId, createFailedWebhookValidationOpts)
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

func (oc *OLAPControllerImpl) processPayloadExternalCutovers(ctx context.Context) func() {
	return func() {
		ctx, span := telemetry.NewSpan(ctx, "OLAPControllerImpl.processPayloadExternalCutovers")
		defer span.End()

		oc.l.Debug().Msgf("payload external cutover: processing external cutover payloads")

		p := oc.repo.Payloads()
		err := oc.repo.OLAP().ProcessOLAPPayloadCutovers(ctx, p.ExternalStoreEnabled(), p.InlineStoreTTL(), p.ExternalCutoverBatchSize(), p.ExternalCutoverNumConcurrentOffloads())

		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "could not process external cutover payloads")
			oc.l.Error().Err(err).Msg("could not process external cutover payloads")
		}
	}
}
