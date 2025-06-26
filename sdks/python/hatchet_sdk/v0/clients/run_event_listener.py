import asyncio
import json
from typing import AsyncGenerator

import grpc

from hatchet_sdk.contracts.dispatcher_pb2 import (
    RESOURCE_TYPE_STEP_RUN,
    RESOURCE_TYPE_WORKFLOW_RUN,
    ResourceEventType,
    SubscribeToWorkflowEventsRequest,
    WorkflowEvent,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.v0.connection import new_conn

from ..loader import ClientConfig
from ..metadata import get_metadata

DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5


class StepRunEventType:
    STEP_RUN_EVENT_TYPE_STARTED = "STEP_RUN_EVENT_TYPE_STARTED"
    STEP_RUN_EVENT_TYPE_COMPLETED = "STEP_RUN_EVENT_TYPE_COMPLETED"
    STEP_RUN_EVENT_TYPE_FAILED = "STEP_RUN_EVENT_TYPE_FAILED"
    STEP_RUN_EVENT_TYPE_CANCELLED = "STEP_RUN_EVENT_TYPE_CANCELLED"
    STEP_RUN_EVENT_TYPE_TIMED_OUT = "STEP_RUN_EVENT_TYPE_TIMED_OUT"
    STEP_RUN_EVENT_TYPE_STREAM = "STEP_RUN_EVENT_TYPE_STREAM"


class WorkflowRunEventType:
    WORKFLOW_RUN_EVENT_TYPE_STARTED = "WORKFLOW_RUN_EVENT_TYPE_STARTED"
    WORKFLOW_RUN_EVENT_TYPE_COMPLETED = "WORKFLOW_RUN_EVENT_TYPE_COMPLETED"
    WORKFLOW_RUN_EVENT_TYPE_FAILED = "WORKFLOW_RUN_EVENT_TYPE_FAILED"
    WORKFLOW_RUN_EVENT_TYPE_CANCELLED = "WORKFLOW_RUN_EVENT_TYPE_CANCELLED"
    WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT = "WORKFLOW_RUN_EVENT_TYPE_TIMED_OUT"


step_run_event_type_mapping = {
    ResourceEventType.RESOURCE_EVENT_TYPE_STARTED: StepRunEventType.STEP_RUN_EVENT_TYPE_STARTED,
    ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED: StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED,
    ResourceEventType.RESOURCE_EVENT_TYPE_FAILED: StepRunEventType.STEP_RUN_EVENT_TYPE_FAILED,
    ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED: StepRunEventType.STEP_RUN_EVENT_TYPE_CANCELLED,
    ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT: StepRunEventType.STEP_RUN_EVENT_TYPE_TIMED_OUT,
    ResourceEventType.RESOURCE_EVENT_TYPE_STREAM: StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM,
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


def new_listener(config: ClientConfig):
    return RunEventListenerClient(config=config)


class RunEventListener:

    workflow_run_id: str = None
    additional_meta_kv: tuple[str, str] = None

    def __init__(self, client: DispatcherStub, token: str):
        self.client = client
        self.stop_signal = False
        self.token = token

    @classmethod
    def for_run_id(cls, workflow_run_id: str, client: DispatcherStub, token: str):
        listener = RunEventListener(client, token)
        listener.workflow_run_id = workflow_run_id
        return listener

    @classmethod
    def for_additional_meta(
        cls, key: str, value: str, client: DispatcherStub, token: str
    ):
        listener = RunEventListener(client, token)
        listener.additional_meta_kv = (key, value)
        return listener

    def abort(self):
        self.stop_signal = True

    def __aiter__(self):
        return self._generator()

    async def __anext__(self):
        return await self._generator().__anext__()

    def __iter__(self):
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError as e:
            if str(e).startswith("There is no current event loop in thread"):
                loop = asyncio.new_event_loop()
                asyncio.set_event_loop(loop)
            else:
                raise e

        async_iter = self.__aiter__()

        while True:
            try:
                future = asyncio.ensure_future(async_iter.__anext__())
                yield loop.run_until_complete(future)
            except StopAsyncIteration:
                break
            except Exception as e:
                print(f"Error in synchronous iterator: {e}")
                break

    async def _generator(self) -> AsyncGenerator[StepRunEvent, None]:
        while True:
            if self.stop_signal:
                listener = None
                break

            listener = await self.retry_subscribe()

            try:
                async for workflow_event in listener:
                    eventType = None
                    if workflow_event.resourceType == RESOURCE_TYPE_STEP_RUN:
                        if workflow_event.eventType in step_run_event_type_mapping:
                            eventType = step_run_event_type_mapping[
                                workflow_event.eventType
                            ]
                        else:
                            raise Exception(
                                f"Unknown event type: {workflow_event.eventType}"
                            )
                        payload = None

                        try:
                            if workflow_event.eventPayload:
                                payload = json.loads(workflow_event.eventPayload)
                        except Exception as e:
                            payload = workflow_event.eventPayload
                            pass

                        yield StepRunEvent(type=eventType, payload=payload)
                    elif workflow_event.resourceType == RESOURCE_TYPE_WORKFLOW_RUN:
                        if workflow_event.eventType in workflow_run_event_type_mapping:
                            eventType = workflow_run_event_type_mapping[
                                workflow_event.eventType
                            ]
                        else:
                            raise Exception(
                                f"Unknown event type: {workflow_event.eventType}"
                            )

                        payload = None

                        try:
                            if workflow_event.eventPayload:
                                payload = json.loads(workflow_event.eventPayload)
                        except Exception as e:
                            pass

                        yield StepRunEvent(type=eventType, payload=payload)

                    if workflow_event.hangup:
                        listener = None
                        break

                break
            except grpc.RpcError as e:
                # Handle different types of errors
                if e.code() == grpc.StatusCode.CANCELLED:
                    # Context cancelled, unsubscribe and close
                    break
                elif e.code() == grpc.StatusCode.UNAVAILABLE:
                    # Retry logic
                    # logger.info("Could not connect to Hatchet, retrying...")
                    listener = await self.retry_subscribe()
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    # logger.info("Deadline exceeded, retrying subscription")
                    continue
                else:
                    # Unknown error, report and break
                    # logger.error(f"Failed to receive message: {e}")
                    break
                # Raise StopAsyncIteration to properly end the generator

    async def retry_subscribe(self):
        retries = 0

        while retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    await asyncio.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)

                if self.workflow_run_id is not None:
                    return self.client.SubscribeToWorkflowEvents(
                        SubscribeToWorkflowEventsRequest(
                            workflowRunId=self.workflow_run_id,
                        ),
                        metadata=get_metadata(self.token),
                    )
                elif self.additional_meta_kv is not None:
                    return self.client.SubscribeToWorkflowEvents(
                        SubscribeToWorkflowEventsRequest(
                            additionalMetaKey=self.additional_meta_kv[0],
                            additionalMetaValue=self.additional_meta_kv[1],
                        ),
                        metadata=get_metadata(self.token),
                    )
                else:
                    raise Exception("no listener method provided")

            except grpc.RpcError as e:
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError(f"gRPC error: {e}")


class RunEventListenerClient:
    def __init__(self, config: ClientConfig):
        self.token = config.token
        self.config = config
        self.client: DispatcherStub = None

    def stream_by_run_id(self, workflow_run_id: str):
        return self.stream(workflow_run_id)

    def stream(self, workflow_run_id: str):
        if not isinstance(workflow_run_id, str) and hasattr(workflow_run_id, "__str__"):
            workflow_run_id = str(workflow_run_id)

        if not self.client:
            aio_conn = new_conn(self.config, True)
            self.client = DispatcherStub(aio_conn)

        return RunEventListener.for_run_id(workflow_run_id, self.client, self.token)

    def stream_by_additional_metadata(self, key: str, value: str):
        if not self.client:
            aio_conn = new_conn(self.config, True)
            self.client = DispatcherStub(aio_conn)

        return RunEventListener.for_additional_meta(key, value, self.client, self.token)

    async def on(self, workflow_run_id: str, handler: callable = None):
        async for event in self.stream(workflow_run_id):
            # call the handler if provided
            if handler:
                handler(event)
