package dispatcher

import (
	"context"
	"fmt"
	"time"

	"github.com/hatchet-dev/hatchet/internal/datautils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/internal/repository/prisma/db"
	"github.com/hatchet-dev/hatchet/internal/services/dispatcher/contracts"
	"github.com/hatchet-dev/hatchet/internal/services/shared/tasktypes"
	"github.com/hatchet-dev/hatchet/internal/taskqueue"
	"github.com/hatchet-dev/hatchet/internal/telemetry"
)

func (d *DispatcherImpl) GetWorker(workerId string) (*subscribedWorker, error) {
	workerInt, ok := d.workers.Load(workerId)
	if !ok {
		return nil, fmt.Errorf("worker with id %s not found", workerId)
	}

	worker, ok := workerInt.(subscribedWorker)

	if !ok {
		return nil, fmt.Errorf("failed to cast worker with id %s to subscribedWorker", workerId)
	}

	return &worker, nil
}

type subscribedWorker struct {
	// stream is the server side of the RPC stream
	stream contracts.Dispatcher_ListenServer

	// finished is used to signal closure of a client subscribing goroutine
	finished chan<- bool
}

func (worker *subscribedWorker) StartStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *db.StepRunModel,
) error {
	ctx, span := telemetry.NewSpan(ctx, "start-step-run")
	defer span.End()

	inputBytes := []byte{}
	inputType, ok := stepRun.Input()

	if ok {
		var err error
		inputBytes, err = inputType.MarshalJSON()

		if err != nil {
			return err
		}
	}

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:      tenantId,
		JobId:         stepRun.Step().JobID,
		JobName:       stepRun.Step().Job().Name,
		JobRunId:      stepRun.JobRunID,
		StepId:        stepRun.StepID,
		StepRunId:     stepRun.ID,
		ActionType:    contracts.ActionType_START_STEP_RUN,
		ActionId:      stepRun.Step().ActionID,
		ActionPayload: string(inputBytes),
	})
}

func (worker *subscribedWorker) CancelStepRun(
	ctx context.Context,
	tenantId string,
	stepRun *db.StepRunModel,
) error {
	ctx, span := telemetry.NewSpan(ctx, "cancel-step-run")
	defer span.End()

	return worker.stream.Send(&contracts.AssignedAction{
		TenantId:   tenantId,
		JobId:      stepRun.Step().JobID,
		JobName:    stepRun.Step().Job().Name,
		JobRunId:   stepRun.JobRunID,
		StepId:     stepRun.StepID,
		StepRunId:  stepRun.ID,
		ActionType: contracts.ActionType_CANCEL_STEP_RUN,
	})
}

func (s *DispatcherImpl) Register(ctx context.Context, request *contracts.WorkerRegisterRequest) (*contracts.WorkerRegisterResponse, error) {
	// TODO: auth checks to make sure the worker is allowed to register for this tenant

	s.l.Debug().Msgf("Received register request from ID %s with actions %v", request.WorkerName, request.Actions)

	svcs := request.Services

	if len(svcs) == 0 {
		svcs = []string{"default"}
	}

	// create a worker in the database
	worker, err := s.repo.Worker().CreateNewWorker(request.TenantId, &repository.CreateWorkerOpts{
		DispatcherId: s.dispatcherId,
		Name:         request.WorkerName,
		Actions:      request.Actions,
		Services:     svcs,
	})

	if err != nil {
		s.l.Error().Err(err).Msgf("could not create worker for tenant %s", request.TenantId)
		return nil, err
	}

	s.l.Debug().Msgf("Registered worker with ID: %s", worker.ID)

	// return the worker id to the worker
	return &contracts.WorkerRegisterResponse{
		TenantId:   worker.TenantID,
		WorkerId:   worker.ID,
		WorkerName: worker.Name,
	}, nil
}

// Subscribe handles a subscribe request from a client
func (s *DispatcherImpl) Listen(request *contracts.WorkerListenRequest, stream contracts.Dispatcher_ListenServer) error {
	s.l.Debug().Msgf("Received subscribe request from ID: %s", request.WorkerId)

	worker, err := s.repo.Worker().GetWorkerById(request.WorkerId)

	if err != nil {
		s.l.Error().Err(err).Msgf("could not get worker %s", request.WorkerId)
		return err
	}

	// check the worker's dispatcher against the current dispatcher. if they don't match, then update the worker
	if worker.DispatcherID != s.dispatcherId {
		_, err = s.repo.Worker().UpdateWorker(request.TenantId, request.WorkerId, &repository.UpdateWorkerOpts{
			DispatcherId: &s.dispatcherId,
		})

		if err != nil {
			s.l.Error().Err(err).Msgf("could not update worker %s dispatcher", request.WorkerId)
			return err
		}
	}

	fin := make(chan bool)

	s.workers.Store(request.WorkerId, subscribedWorker{stream: stream, finished: fin})

	defer func() {
		s.workers.Delete(request.WorkerId)

		inactive := db.WorkerStatusInactive

		_, err := s.repo.Worker().UpdateWorker(request.TenantId, request.WorkerId, &repository.UpdateWorkerOpts{
			Status: &inactive,
		})

		if err != nil {
			s.l.Error().Err(err).Msgf("could not update worker %s status to inactive", request.WorkerId)
		}
	}()

	ctx := stream.Context()

	// update the worker with a last heartbeat time every 5 seconds as long as the worker is connected
	go func() {
		timer := time.NewTicker(100 * time.Millisecond)

		// set the last heartbeat to 6 seconds ago so the first heartbeat is sent immediately
		lastHeartbeat := time.Now().UTC().Add(-6 * time.Second)
		defer timer.Stop()

		for {
			select {
			case <-ctx.Done():
				s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)
				return
			case <-timer.C:
				if now := time.Now().UTC(); lastHeartbeat.Add(5 * time.Second).Before(now) {
					s.l.Debug().Msgf("updating worker %s heartbeat", request.WorkerId)

					_, err := s.repo.Worker().UpdateWorker(request.TenantId, request.WorkerId, &repository.UpdateWorkerOpts{
						LastHeartbeatAt: &now,
					})

					if err != nil {
						s.l.Error().Err(err).Msgf("could not update worker %s heartbeat", request.WorkerId)
						return
					}

					lastHeartbeat = time.Now().UTC()
				}
			}
		}
	}()

	// Keep the connection alive for sending messages
	for {
		select {
		case <-fin:
			s.l.Debug().Msgf("closing stream for worker id: %s", request.WorkerId)
			return nil
		case <-ctx.Done():
			s.l.Debug().Msgf("worker id %s has disconnected", request.WorkerId)
			return nil
		}
	}
}

func (s *DispatcherImpl) SendActionEvent(ctx context.Context, request *contracts.ActionEvent) (*contracts.ActionEventResponse, error) {
	// TODO: auth checks to make sure the worker is allowed to send an action event for this tenant

	switch request.EventType {
	case contracts.ActionEventType_STEP_EVENT_TYPE_STARTED:
		return s.handleStepRunStarted(ctx, request)
	case contracts.ActionEventType_STEP_EVENT_TYPE_COMPLETED:
		return s.handleStepRunCompleted(ctx, request)
	case contracts.ActionEventType_STEP_EVENT_TYPE_FAILED:
		return s.handleStepRunFailed(ctx, request)
	}

	return nil, fmt.Errorf("unknown event type %s", request.EventType)
}

func (s *DispatcherImpl) Unsubscribe(ctx context.Context, request *contracts.WorkerUnsubscribeRequest) (*contracts.WorkerUnsubscribeResponse, error) {
	// TODO: auth checks to make sure the worker is allowed to unsubscribe for this tenant
	// no matter what, remove the worker from the connection pool
	defer s.workers.Delete(request.WorkerId)

	err := s.repo.Worker().DeleteWorker(request.TenantId, request.WorkerId)

	if err != nil {
		return nil, err
	}

	return &contracts.WorkerUnsubscribeResponse{
		TenantId: request.TenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleStepRunStarted(ctx context.Context, request *contracts.ActionEvent) (*contracts.ActionEventResponse, error) {
	s.l.Debug().Msgf("Received step started event for step run %s", request.StepRunId)

	startedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskPayload{
		StepRunId: request.StepRunId,
		StartedAt: startedAt.Format(time.RFC3339),
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunStartedTaskMetadata{
		TenantId: request.TenantId,
	})

	// send the event to the jobs queue
	err := s.tq.AddTask(ctx, taskqueue.JOB_PROCESSING_QUEUE, &taskqueue.Task{
		ID:       "step-run-started",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: request.TenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleStepRunCompleted(ctx context.Context, request *contracts.ActionEvent) (*contracts.ActionEventResponse, error) {
	s.l.Debug().Msgf("Received step completed event for step run %s", request.StepRunId)

	finishedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskPayload{
		StepRunId:      request.StepRunId,
		FinishedAt:     finishedAt.Format(time.RFC3339),
		StepOutputData: request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFinishedTaskMetadata{
		TenantId: request.TenantId,
	})

	// send the event to the jobs queue
	err := s.tq.AddTask(ctx, taskqueue.JOB_PROCESSING_QUEUE, &taskqueue.Task{
		ID:       "step-run-finished",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: request.TenantId,
		WorkerId: request.WorkerId,
	}, nil
}

func (s *DispatcherImpl) handleStepRunFailed(ctx context.Context, request *contracts.ActionEvent) (*contracts.ActionEventResponse, error) {
	s.l.Debug().Msgf("Received step failed event for step run %s", request.StepRunId)

	failedAt := request.EventTimestamp.AsTime()

	payload, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskPayload{
		StepRunId: request.StepRunId,
		FailedAt:  failedAt.Format(time.RFC3339),
		Error:     request.EventPayload,
	})

	metadata, _ := datautils.ToJSONMap(tasktypes.StepRunFailedTaskMetadata{
		TenantId: request.TenantId,
	})

	// send the event to the jobs queue
	err := s.tq.AddTask(ctx, taskqueue.JOB_PROCESSING_QUEUE, &taskqueue.Task{
		ID:       "step-run-failed",
		Queue:    taskqueue.JOB_PROCESSING_QUEUE,
		Payload:  payload,
		Metadata: metadata,
	})

	if err != nil {
		return nil, err
	}

	return &contracts.ActionEventResponse{
		TenantId: request.TenantId,
		WorkerId: request.WorkerId,
	}, nil
}
