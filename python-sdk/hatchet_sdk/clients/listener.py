from typing import List
import grpc
from ..dispatcher_pb2_grpc import DispatcherStub

from ..dispatcher_pb2 import SubscribeToWorkflowEventsRequest, ResourceEventType, WorkflowEvent, RESOURCE_TYPE_STEP_RUN, RESOURCE_TYPE_WORKFLOW_RUN
from ..loader import ClientConfig
from ..metadata import get_metadata
import json
import time


DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5


class StepRunEventType:
    STEP_RUN_EVENT_TYPE_STARTED = 'STEP_RUN_EVENT_TYPE_STARTED'
    STEP_RUN_EVENT_TYPE_COMPLETED = 'STEP_RUN_EVENT_TYPE_COMPLETED'
    STEP_RUN_EVENT_TYPE_FAILED = 'STEP_RUN_EVENT_TYPE_FAILED'
    STEP_RUN_EVENT_TYPE_CANCELLED = 'STEP_RUN_EVENT_TYPE_CANCELLED'
    STEP_RUN_EVENT_TYPE_TIMED_OUT = 'STEP_RUN_EVENT_TYPE_TIMED_OUT'

class WorkflowRunEventType:
    WORKFLOW_RUN_EVENT_TYPE_STARTED = 'WORKFLOW_RUN_EVENT_TYPE_STARTED'
    WORKFLOW_RUN_EVENT_TYPE_COMPLETED = 'WORKFLOW_RUN_EVENT_TYPE_COMPLETED'
    WORKFLOW_RUN_EVENT_TYPE_FAILED = 'WORKFLOW_RUN_EVENT_TYPE_FAILED'
    WORKFLOW_RUN_EVENT_TYPE_CANCELLED = 'WORKFLOW_RUN_EVENT_TYPE_CANCELLED'
    WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT = 'WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT'

step_run_event_type_mapping = {
    ResourceEventType.RESOURCE_EVENT_TYPE_STARTED: StepRunEventType.STEP_RUN_EVENT_TYPE_STARTED,
    ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED: StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED,
    ResourceEventType.RESOURCE_EVENT_TYPE_FAILED: StepRunEventType.STEP_RUN_EVENT_TYPE_FAILED,
    ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED: StepRunEventType.STEP_RUN_EVENT_TYPE_CANCELLED,
    ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT: StepRunEventType.STEP_RUN_EVENT_TYPE_TIMED_OUT,
}

workflow_run_event_type_mapping = {
    ResourceEventType.RESOURCE_EVENT_TYPE_STARTED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_STARTED,
    ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_COMPLETED,
    ResourceEventType.RESOURCE_EVENT_TYPE_FAILED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_FAILED,
    ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_CANCELLED,
    ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT: WorkflowRunEventType.WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT,
}

class StepRunEvent:
    def __init__(self, type: StepRunEventType, payload: str):
        self.type = type
        self.payload = payload


def new_listener(conn, config: ClientConfig):
    return ListenerClientImpl(
        client=DispatcherStub(conn),
        token=config.token,
    )


class HatchetListener:
    def __init__(self, client: DispatcherStub, workflow_run_id: str, token: str):
        self.client = client
        self.stop_signal = False
        self.workflow_run_id = workflow_run_id
        self.token = token

    def __iter__(self):
        return self._generator()

    def abort(self):
        self.stop_signal = True

    def _generator(self, stop_step: str = None) -> List[StepRunEvent]:
        listener = self.retry_subscribe()
        while listener:
            if self.stop_signal:
                listener = None
                break

            try:
                for workflow_event in listener:
                    eventType = None

                    if workflow_event.resourceType == RESOURCE_TYPE_STEP_RUN:
                        if workflow_event.eventType in step_run_event_type_mapping:
                            eventType = step_run_event_type_mapping[workflow_event.eventType]
                        else:
                            raise Exception(
                                f"Unknown event type: {workflow_event.eventType}")
                        payload = None

                        try:
                            if workflow_event.eventPayload:
                                payload = json.loads(workflow_event.eventPayload)
                        except Exception as e:
                            pass

                        # call the handler
                        event = StepRunEvent(type=eventType, payload=payload)
                        yield event
                    elif workflow_event.resourceType == RESOURCE_TYPE_WORKFLOW_RUN:
                        if workflow_event.eventType in workflow_run_event_type_mapping:
                            eventType = workflow_run_event_type_mapping[workflow_event.eventType]
                        else:
                            raise Exception(
                                f"Unknown event type: {workflow_event.eventType}")
                        
                        payload = None
                        
                        try:
                            if workflow_event.eventPayload:
                                payload = json.loads(workflow_event.eventPayload)
                        except Exception as e:
                            pass
                        
                    if workflow_event.hangup:
                        listener = None
                        print('hangup stopping listener...')
                        break

            except grpc.RpcError as e:
                # Handle different types of errors
                if e.code() == grpc.StatusCode.CANCELLED:
                    # Context cancelled, unsubscribe and close
                    break
                elif e.code() == grpc.StatusCode.UNAVAILABLE:
                    # Retry logic
                    # logger.info("Could not connect to Hatchet, retrying...")
                    listener = self.retry_subscribe()
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    # logger.info("Deadline exceeded, retrying subscription")
                    continue
                else:
                    # Unknown error, report and break
                    # logger.error(f"Failed to receive message: {e}")
                    break

    def retry_subscribe(self):
        retries = 0

        while retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)

                listener = self.client.SubscribeToWorkflowEvents(
                    SubscribeToWorkflowEventsRequest(
                        workflowRunId=self.workflow_run_id,
                    ), metadata=get_metadata(self.token))
                return listener
            except grpc.RpcError as e:
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError(f"gRPC error: {e}")


class ListenerClientImpl:
    def __init__(self, client: DispatcherStub, token: str):
        self.client = client
        self.token = token

    def stream(self, workflow_run_id: str):
        return HatchetListener(self.client, workflow_run_id, self.token)

    def on(self, workflow_run_id: str, handler: callable = None):
        for event in self.stream(workflow_run_id):
            # call the handler if provided
            if handler:
                handler(event)
