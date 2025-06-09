import asyncio
from collections.abc import AsyncGenerator, Callable, Generator
from enum import Enum
from queue import Empty, Queue
from threading import Thread
from typing import Any, Literal, TypeVar, cast

import grpc
from pydantic import BaseModel

from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    RESOURCE_TYPE_STEP_RUN,
    RESOURCE_TYPE_WORKFLOW_RUN,
    ResourceEventType,
    SubscribeToWorkflowEventsRequest,
    WorkflowEvent,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.metadata import get_metadata

DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5


class StepRunEventType(str, Enum):
    STEP_RUN_EVENT_TYPE_STARTED = "STEP_RUN_EVENT_TYPE_STARTED"
    STEP_RUN_EVENT_TYPE_COMPLETED = "STEP_RUN_EVENT_TYPE_COMPLETED"
    STEP_RUN_EVENT_TYPE_FAILED = "STEP_RUN_EVENT_TYPE_FAILED"
    STEP_RUN_EVENT_TYPE_CANCELLED = "STEP_RUN_EVENT_TYPE_CANCELLED"
    STEP_RUN_EVENT_TYPE_TIMED_OUT = "STEP_RUN_EVENT_TYPE_TIMED_OUT"
    STEP_RUN_EVENT_TYPE_STREAM = "STEP_RUN_EVENT_TYPE_STREAM"


class WorkflowRunEventType(str, Enum):
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

T = TypeVar("T")


class StepRunEvent(BaseModel):
    type: StepRunEventType
    payload: str


class RunEventListener:
    def __init__(
        self,
        config: ClientConfig,
        workflow_run_id: str | None = None,
        additional_meta_kv: tuple[str, str] | None = None,
    ):
        self.config = config
        self.stop_signal = False

        self.workflow_run_id = workflow_run_id
        self.additional_meta_kv = additional_meta_kv

        ## IMPORTANT: This needs to be created lazily so we don't require
        ## an event loop to instantiate the client.
        self.client: DispatcherStub | None = None

    def abort(self) -> None:
        self.stop_signal = True

    def __aiter__(self) -> AsyncGenerator[StepRunEvent, None]:
        return self._generator()

    async def __anext__(self) -> StepRunEvent:
        return await self._generator().__anext__()

    def async_to_sync_thread(
        self, async_iter: AsyncGenerator[T, None]
    ) -> Generator[T, None, None]:
        q = Queue[T | Literal["DONE"]]()
        done_sentinel: Literal["DONE"] = "DONE"

        def runner() -> None:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

            async def consume() -> None:
                try:
                    async for item in async_iter:
                        q.put(item)
                finally:
                    q.put(done_sentinel)

            try:
                loop.run_until_complete(consume())
            finally:
                loop.stop()
                loop.close()

        thread = Thread(target=runner)
        thread.start()

        while True:
            try:
                item = q.get(timeout=1)
                if item == "DONE":
                    break
                yield item
            except Empty:
                continue

        thread.join()

    def __iter__(self) -> Generator[StepRunEvent, None, None]:
        yield from self.async_to_sync_thread(self.__aiter__())

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

                        yield StepRunEvent(
                            type=eventType, payload=workflow_event.eventPayload
                        )
                    elif workflow_event.resourceType == RESOURCE_TYPE_WORKFLOW_RUN:
                        if workflow_event.eventType in step_run_event_type_mapping:
                            workflowRunEventType = step_run_event_type_mapping[
                                workflow_event.eventType
                            ]
                        else:
                            raise Exception(
                                f"Unknown event type: {workflow_event.eventType}"
                            )

                        yield StepRunEvent(
                            type=workflowRunEventType,
                            payload=workflow_event.eventPayload,
                        )

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

    async def retry_subscribe(self) -> AsyncGenerator[WorkflowEvent, None]:
        retries = 0

        if self.client is None:
            aio_conn = new_conn(self.config, True)
            self.client = DispatcherStub(aio_conn)

        while retries < DEFAULT_ACTION_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    await asyncio.sleep(DEFAULT_ACTION_LISTENER_RETRY_INTERVAL)

                if self.workflow_run_id is not None:
                    return cast(
                        AsyncGenerator[WorkflowEvent, None],
                        self.client.SubscribeToWorkflowEvents(
                            SubscribeToWorkflowEventsRequest(
                                workflowRunId=self.workflow_run_id,
                            ),
                            metadata=get_metadata(self.config.token),
                        ),
                    )
                if self.additional_meta_kv is not None:
                    return cast(
                        AsyncGenerator[WorkflowEvent, None],
                        self.client.SubscribeToWorkflowEvents(
                            SubscribeToWorkflowEventsRequest(
                                additionalMetaKey=self.additional_meta_kv[0],
                                additionalMetaValue=self.additional_meta_kv[1],
                            ),
                            metadata=get_metadata(self.config.token),
                        ),
                    )
                raise Exception("no listener method provided")

            except grpc.RpcError as e:  # noqa: PERF203
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError("gRPC error") from e

        raise Exception("Failed to subscribe to workflow events")


class RunEventListenerClient:
    def __init__(self, config: ClientConfig):
        self.config = config

    def stream_by_run_id(self, workflow_run_id: str) -> RunEventListener:
        return self.stream(workflow_run_id)

    def stream(self, workflow_run_id: str) -> RunEventListener:
        return RunEventListener(config=self.config, workflow_run_id=workflow_run_id)

    def stream_by_additional_metadata(self, key: str, value: str) -> RunEventListener:
        return RunEventListener(config=self.config, additional_meta_kv=(key, value))

    async def on(
        self, workflow_run_id: str, handler: Callable[[StepRunEvent], Any] | None = None
    ) -> None:
        async for event in self.stream(workflow_run_id):
            # call the handler if provided
            if handler:
                handler(event)
