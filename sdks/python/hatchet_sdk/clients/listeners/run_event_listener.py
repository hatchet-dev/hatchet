import asyncio
from collections.abc import AsyncGenerator
from enum import Enum
from typing import TypeVar, cast

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
from hatchet_sdk.utils.api_auth import create_authorization_header

DEFAULT_ACTION_LISTENER_RETRY_INTERVAL = 5  # seconds
DEFAULT_ACTION_LISTENER_RETRY_COUNT = 5


class TaskRunEventType(str, Enum):
    STARTED = "STARTED"
    COMPLETED = "COMPLETED"
    FAILED = "FAILED"
    CANCELLED = "CANCELLED"
    TIMED_OUT = "TIMED_OUT"
    STREAM = "STREAM"


task_run_event_type_mapping = {
    ResourceEventType.RESOURCE_EVENT_TYPE_STARTED: TaskRunEventType.STARTED,
    ResourceEventType.RESOURCE_EVENT_TYPE_COMPLETED: TaskRunEventType.COMPLETED,
    ResourceEventType.RESOURCE_EVENT_TYPE_FAILED: TaskRunEventType.FAILED,
    ResourceEventType.RESOURCE_EVENT_TYPE_CANCELLED: TaskRunEventType.CANCELLED,
    ResourceEventType.RESOURCE_EVENT_TYPE_TIMED_OUT: TaskRunEventType.TIMED_OUT,
    ResourceEventType.RESOURCE_EVENT_TYPE_STREAM: TaskRunEventType.STREAM,
}

T = TypeVar("T")


class TaskRunEvent(BaseModel):
    type: TaskRunEventType
    payload: str


class RunEventListener:
    def __init__(
        self,
        config: ClientConfig,
        workflow_run_id: str | None = None,
        additional_meta_kv: tuple[str, str] | None = None,
    ) -> None:
        self.config = config
        self.stop_signal = False

        self.workflow_run_id = workflow_run_id
        self.additional_meta_kv = additional_meta_kv

        ## IMPORTANT: This needs to be created lazily so we don't require
        ## an event loop to instantiate the client.
        self.client: DispatcherStub | None = None

    def __aiter__(self) -> AsyncGenerator[TaskRunEvent, None]:
        return self._generator()

    async def __anext__(self) -> TaskRunEvent:
        return await self._generator().__anext__()

    async def _generator(self) -> AsyncGenerator[TaskRunEvent, None]:
        while True:
            if self.stop_signal:
                listener = None
                break

            listener = await self.retry_subscribe()

            try:
                async for workflow_event in listener:
                    if workflow_event.resource_type in (
                        RESOURCE_TYPE_STEP_RUN,
                        RESOURCE_TYPE_WORKFLOW_RUN,
                    ):
                        if workflow_event.event_type not in task_run_event_type_mapping:
                            raise Exception(
                                f"Unknown event type: {workflow_event.event_type}"
                            )

                        yield TaskRunEvent(
                            type=task_run_event_type_mapping[workflow_event.event_type],
                            payload=workflow_event.event_payload,
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
                    listener = await self.retry_subscribe()
                elif e.code() == grpc.StatusCode.DEADLINE_EXCEEDED:
                    continue
                else:
                    # Unknown error, report and break
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
                        "AsyncGenerator[WorkflowEvent, None]",
                        self.client.SubscribeToWorkflowEvents(
                            SubscribeToWorkflowEventsRequest(
                                workflow_run_id=self.workflow_run_id,
                            ),
                            metadata=create_authorization_header(self.config.token),
                        ),
                    )
                if self.additional_meta_kv is not None:
                    return cast(
                        "AsyncGenerator[WorkflowEvent, None]",
                        self.client.SubscribeToWorkflowEvents(
                            SubscribeToWorkflowEventsRequest(
                                additional_meta_key=self.additional_meta_kv[0],
                                additional_meta_value=self.additional_meta_kv[1],
                            ),
                            metadata=create_authorization_header(self.config.token),
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
    def __init__(self, config: ClientConfig) -> None:
        self.config = config

    def stream(self, workflow_run_id: str) -> RunEventListener:
        return RunEventListener(config=self.config, workflow_run_id=workflow_run_id)
