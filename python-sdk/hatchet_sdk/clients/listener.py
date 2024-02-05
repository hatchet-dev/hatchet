from typing import List
import grpc
from ..dispatcher_pb2_grpc import DispatcherStub

from ..dispatcher_pb2 import SubscribeToWorkflowEventsRequest, WorkflowEvent
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


class StepRunEvent:
    def __init__(self, type: StepRunEventType, payload: str):
        self.type = type
        self.payload = payload


def new_listener(conn, config: ClientConfig):
    return ListenerClientImpl(
        client=DispatcherStub(conn),
        token=config.token,
    )


class ListenerClientImpl:
    def __init__(self, client: DispatcherStub, token):
        self.client = client
        self.token = token

    def on(self, workflowRunId: str, handler: callable):
        listener = self.retry_subscribe(workflowRunId)

        print('x', listener)

        while True:
            try:
                for workflow_event in listener:
                    print('y', workflow_event)
                    # TODO handler()

            except grpc.RpcError as e:
                # Handle different types of errors
                if e.code() == grpc.StatusCode.CANCELLED:
                    # Context cancelled, unsubscribe and close
                    break
                elif e.code() == grpc.StatusCode.UNAVAILABLE:
                    # Retry logic
                    logger.info("Could not connect to Hatchet, retrying...")
                    listener = self.retry_subscribe(workflowRunId)
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    logger.info("Deadline exceeded, retrying subscription")
                    continue
                else:
                    # Unknown error, report and break
                    logger.error(f"Failed to receive message: {e}")
                    break

    def retry_subscribe(self, workflowRunId: str):
        retries = 0

        while retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    time.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)

                listener = self.client.SubscribeToWorkflowEvents(
                    SubscribeToWorkflowEventsRequest(
                        workflowRunId=workflowRunId,
                    ), metadata=get_metadata(self.token))
                return listener
            except grpc.RpcError as e:
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError(f"gRPC error: {e}")
