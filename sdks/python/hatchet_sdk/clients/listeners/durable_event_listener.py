import json
from collections.abc import AsyncIterator
from typing import Any, Literal, cast

import grpc
import grpc.aio
from pydantic import BaseModel, ConfigDict

from hatchet_sdk.clients.listeners.pooled_listener import PooledListener
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.conditions import Condition, SleepCondition, UserEventCondition
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    CreateDurableEventLogRequest,
    DurableEvent,
    GetDurableEventLogRequest,
    GetDurableEventLogResponse,
    ListenForDurableEventRequest,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    RegisterDurableEventRequest as RegisterDurableEventRequestProto,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2_grpc import V1DispatcherStub
from hatchet_sdk.contracts.v1.shared.condition_pb2 import DurableEventListenerConditions
from hatchet_sdk.metadata import get_metadata

DEFAULT_DURABLE_EVENT_LISTENER_RETRY_INTERVAL = 3  # seconds
DEFAULT_DURABLE_EVENT_LISTENER_RETRY_COUNT = 5
DEFAULT_DURABLE_EVENT_LISTENER_INTERRUPT_INTERVAL = 1800  # 30 minutes


class RegisterDurableEventRequest(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    task_id: str
    signal_key: str
    conditions: list[Condition]
    config: ClientConfig

    def to_proto(self) -> RegisterDurableEventRequestProto:
        return RegisterDurableEventRequestProto(
            task_id=self.task_id,
            signal_key=self.signal_key,
            conditions=DurableEventListenerConditions(
                sleep_conditions=[
                    c.to_proto(self.config)
                    for c in self.conditions
                    if isinstance(c, SleepCondition)
                ],
                user_event_conditions=[
                    c.to_proto(self.config)
                    for c in self.conditions
                    if isinstance(c, UserEventCondition)
                ],
            ),
        )


class ParsedKey(BaseModel):
    task_id: str
    signal_key: str


class DurableEventListener(
    PooledListener[ListenForDurableEventRequest, DurableEvent, V1DispatcherStub]
):
    def _generate_key(self, task_id: str, signal_key: str) -> str:
        return task_id + ":" + signal_key

    def generate_key(self, response: DurableEvent) -> str:
        return self._generate_key(
            task_id=response.task_id,
            signal_key=response.signal_key,
        )

    def parse_key(self, key: str) -> ParsedKey:
        task_id, signal_key = key.split(":", maxsplit=1)

        return ParsedKey(
            task_id=task_id,
            signal_key=signal_key,
        )

    async def create_subscription(
        self,
        request: AsyncIterator[ListenForDurableEventRequest],
        metadata: tuple[tuple[str, str]],
    ) -> grpc.aio.UnaryStreamCall[ListenForDurableEventRequest, DurableEvent]:
        if self.client is None:
            conn = new_conn(self.config, True)
            self.client = V1DispatcherStub(conn)

        return cast(
            grpc.aio.UnaryStreamCall[ListenForDurableEventRequest, DurableEvent],
            self.client.ListenForDurableEvent(
                request,  # type: ignore[arg-type]
                metadata=metadata,
            ),
        )

    def create_request_body(self, item: str) -> ListenForDurableEventRequest:
        key = self.parse_key(item)
        return ListenForDurableEventRequest(
            task_id=key.task_id,
            signal_key=key.signal_key,
        )

    def register_durable_event(
        self, request: RegisterDurableEventRequest
    ) -> Literal[True]:
        conn = new_conn(self.config, True)
        client = V1DispatcherStub(conn)

        register_durable_event = tenacity_retry(
            client.RegisterDurableEvent, self.config.tenacity
        )

        register_durable_event(
            request.to_proto(),
            timeout=5,
            metadata=get_metadata(self.token),
        )

        return True

    def get_durable_event_log(
        self, external_id: str, key: str
    ) -> GetDurableEventLogResponse:
        conn = new_conn(self.config, True)
        client = V1DispatcherStub(conn)

        get_durable_event_log = tenacity_retry(
            client.GetDurableEventLog, self.config.tenacity
        )

        return get_durable_event_log(
            GetDurableEventLogRequest(
                external_id=external_id,
                key=key,
            ),
            timeout=5,
            metadata=get_metadata(self.token),
        )

    def create_durable_event_log(self, external_id: str, key: str, data: bytes) -> None:
        conn = new_conn(self.config, True)
        client = V1DispatcherStub(conn)

        create_durable_event_log = tenacity_retry(
            client.CreateDurableEventLog, self.config.tenacity
        )

        create_durable_event_log(
            CreateDurableEventLogRequest(
                external_id=external_id,
                key=key,
                data=data,
            ),
            timeout=5,
            metadata=get_metadata(self.token),
        )

    async def result(self, task_id: str, signal_key: str) -> dict[str, Any]:
        key = self._generate_key(task_id, signal_key)

        event = await self.subscribe(key)

        return cast(dict[str, Any], json.loads(event.data.decode("utf-8")))
