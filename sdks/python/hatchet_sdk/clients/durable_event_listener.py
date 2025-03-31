import asyncio
import json
from collections.abc import AsyncIterator
from typing import Any, Literal, cast

import grpc
import grpc.aio
from grpc._cython import cygrpc  # type: ignore[attr-defined]
from pydantic import BaseModel, ConfigDict

from hatchet_sdk.clients.event_ts import ThreadSafeEvent, read_with_interrupt
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableEvent,
    ListenForDurableEventRequest,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    RegisterDurableEventRequest as RegisterDurableEventRequestProto,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2_grpc import V1DispatcherStub
from hatchet_sdk.contracts.v1.shared.condition_pb2 import DurableEventListenerConditions
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata
from hatchet_sdk.waits import SleepCondition, UserEventCondition

DEFAULT_DURABLE_EVENT_LISTENER_RETRY_INTERVAL = 3  # seconds
DEFAULT_DURABLE_EVENT_LISTENER_RETRY_COUNT = 5
DEFAULT_DURABLE_EVENT_LISTENER_INTERRUPT_INTERVAL = 1800  # 30 minutes


class _Subscription:
    def __init__(self, id: int, task_id: str, signal_key: str):
        self.id = id
        self.task_id = task_id
        self.signal_key = signal_key
        self.queue: asyncio.Queue[DurableEvent | None] = asyncio.Queue()

    async def __aiter__(self) -> "_Subscription":
        return self

    async def __anext__(self) -> DurableEvent | None:
        return await self.queue.get()

    async def get(self) -> DurableEvent:
        event = await self.queue.get()

        if event is None:
            raise StopAsyncIteration

        return event

    async def put(self, item: DurableEvent) -> None:
        await self.queue.put(item)

    async def close(self) -> None:
        await self.queue.put(None)


class RegisterDurableEventRequest(BaseModel):
    model_config = ConfigDict(arbitrary_types_allowed=True)

    task_id: str
    signal_key: str
    conditions: list[SleepCondition | UserEventCondition]

    def to_proto(self) -> RegisterDurableEventRequestProto:
        return RegisterDurableEventRequestProto(
            task_id=self.task_id,
            signal_key=self.signal_key,
            conditions=DurableEventListenerConditions(
                sleep_conditions=[
                    c.to_pb() for c in self.conditions if isinstance(c, SleepCondition)
                ],
                user_event_conditions=[
                    c.to_pb()
                    for c in self.conditions
                    if isinstance(c, UserEventCondition)
                ],
            ),
        )


class DurableEventListener:
    def __init__(self, config: ClientConfig):
        self.token = config.token
        self.config = config

        # list of all active subscriptions, mapping from a subscription id to a task id and signal key
        self.subscriptions_to_task_id_signal_key: dict[int, tuple[str, str]] = {}

        # task id-signal key tuples mapped to an array of subscription ids
        self.task_id_signal_key_to_subscriptions: dict[tuple[str, str], list[int]] = {}

        self.subscription_counter: int = 0
        self.subscription_counter_lock: asyncio.Lock = asyncio.Lock()

        self.requests: asyncio.Queue[ListenForDurableEventRequest | int] = (
            asyncio.Queue()
        )

        self.listener: (
            grpc.aio.UnaryStreamCall[ListenForDurableEventRequest, DurableEvent] | None
        ) = None
        self.listener_task: asyncio.Task[None] | None = None

        self.curr_requester: int = 0

        self.events: dict[int, _Subscription] = {}

        self.interrupter: asyncio.Task[None] | None = None

    async def _interrupter(self) -> None:
        """
        _interrupter runs in a separate thread and interrupts the listener according to a configurable duration.
        """
        await asyncio.sleep(DEFAULT_DURABLE_EVENT_LISTENER_INTERRUPT_INTERVAL)

        if self.interrupt is not None:
            self.interrupt.set()

    async def _init_producer(self) -> None:
        conn = new_conn(self.config, True)
        client = V1DispatcherStub(conn)

        try:
            if not self.listener:
                while True:
                    try:
                        self.listener = await self._retry_subscribe(client)

                        logger.debug("Workflow run listener connected.")

                        # spawn an interrupter task
                        if self.interrupter is not None and not self.interrupter.done():
                            self.interrupter.cancel()

                        self.interrupter = asyncio.create_task(self._interrupter())

                        while True:
                            self.interrupt = ThreadSafeEvent()
                            if self.listener is None:
                                continue

                            t = asyncio.create_task(
                                read_with_interrupt(self.listener, self.interrupt)
                            )
                            await self.interrupt.wait()

                            if not t.done():
                                logger.warning(
                                    "Interrupted read_with_interrupt task of durable event listener"
                                )

                                t.cancel()
                                self.listener.cancel()

                                await asyncio.sleep(
                                    DEFAULT_DURABLE_EVENT_LISTENER_RETRY_INTERVAL
                                )
                                break

                            event = t.result()

                            if event is cygrpc.EOF:
                                break

                            # get a list of subscriptions for this task-signal pair
                            subscriptions = (
                                self.task_id_signal_key_to_subscriptions.get(
                                    (event.task_id, event.signal_key), []
                                )
                            )

                            for subscription_id in subscriptions:
                                await self.events[subscription_id].put(event)

                    except grpc.RpcError as e:
                        logger.debug(f"grpc error in durable event listener: {e}")
                        await asyncio.sleep(
                            DEFAULT_DURABLE_EVENT_LISTENER_RETRY_INTERVAL
                        )
                        continue

        except Exception as e:
            logger.error(f"Error in durable event listener: {e}")

            self.listener = None

            # close all subscriptions
            for subscription_id in self.events:
                await self.events[subscription_id].close()

            raise e

    async def _request(self) -> AsyncIterator[ListenForDurableEventRequest]:
        self.curr_requester = self.curr_requester + 1

        # replay all existing subscriptions
        for task_id, signal_key in set(
            self.subscriptions_to_task_id_signal_key.values()
        ):
            yield ListenForDurableEventRequest(
                task_id=task_id,
                signal_key=signal_key,
            )

        while True:
            request = await self.requests.get()

            # if the request is an int which matches the current requester, then we should stop
            if request == self.curr_requester:
                break

            # if we've gotten an int that doesn't match the current requester, then we should ignore it
            if isinstance(request, int):
                continue

            yield request
            self.requests.task_done()

    def cleanup_subscription(self, subscription_id: int) -> None:
        task_id_signal_key = self.subscriptions_to_task_id_signal_key[subscription_id]

        if task_id_signal_key in self.task_id_signal_key_to_subscriptions:
            self.task_id_signal_key_to_subscriptions[task_id_signal_key].remove(
                subscription_id
            )

        del self.subscriptions_to_task_id_signal_key[subscription_id]
        del self.events[subscription_id]

    async def subscribe(self, task_id: str, signal_key: str) -> DurableEvent:
        subscription_id: int | None = None

        try:
            # create a new subscription id, place a mutex on the counter
            async with self.subscription_counter_lock:
                self.subscription_counter += 1
                subscription_id = self.subscription_counter

            self.subscriptions_to_task_id_signal_key[subscription_id] = (
                task_id,
                signal_key,
            )

            if (task_id, signal_key) not in self.task_id_signal_key_to_subscriptions:
                self.task_id_signal_key_to_subscriptions[(task_id, signal_key)] = [
                    subscription_id
                ]
            else:
                self.task_id_signal_key_to_subscriptions[(task_id, signal_key)].append(
                    subscription_id
                )

            self.events[subscription_id] = _Subscription(
                subscription_id, task_id, signal_key
            )

            await self.requests.put(
                ListenForDurableEventRequest(
                    task_id=task_id,
                    signal_key=signal_key,
                )
            )

            if not self.listener_task or self.listener_task.done():
                self.listener_task = asyncio.create_task(self._init_producer())

            return await self.events[subscription_id].get()
        except asyncio.CancelledError:
            raise
        finally:
            if subscription_id:
                self.cleanup_subscription(subscription_id)

    async def _retry_subscribe(
        self,
        client: V1DispatcherStub,
    ) -> grpc.aio.UnaryStreamCall[ListenForDurableEventRequest, DurableEvent]:
        retries = 0

        while retries < DEFAULT_DURABLE_EVENT_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    await asyncio.sleep(DEFAULT_DURABLE_EVENT_LISTENER_RETRY_INTERVAL)

                # signal previous async iterator to stop
                if self.curr_requester != 0:
                    self.requests.put_nowait(self.curr_requester)

                return cast(
                    grpc.aio.UnaryStreamCall[
                        ListenForDurableEventRequest, DurableEvent
                    ],
                    client.ListenForDurableEvent(
                        self._request(),  # type: ignore[arg-type]
                        metadata=get_metadata(self.token),
                    ),
                )
            except grpc.RpcError as e:
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError(f"gRPC error: {e}")

        raise ValueError("Failed to connect to durable event listener")

    @tenacity_retry
    def register_durable_event(
        self, request: RegisterDurableEventRequest
    ) -> Literal[True]:
        conn = new_conn(self.config, True)
        client = V1DispatcherStub(conn)

        client.RegisterDurableEvent(
            request.to_proto(),
            timeout=5,
            metadata=get_metadata(self.token),
        )

        return True

    @tenacity_retry
    async def result(self, task_id: str, signal_key: str) -> dict[str, Any]:
        event = await self.subscribe(task_id, signal_key)

        return cast(dict[str, Any], json.loads(event.data.decode("utf-8")))
