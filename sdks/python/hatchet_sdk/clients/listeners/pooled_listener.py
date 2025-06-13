import asyncio
from abc import ABC, abstractmethod
from collections.abc import AsyncIterator
from typing import Generic, Literal, TypeVar

import grpc
import grpc.aio

from hatchet_sdk.clients.event_ts import (
    ThreadSafeEvent,
    UnexpectedEOF,
    read_with_interrupt,
)
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.metadata import get_metadata

DEFAULT_LISTENER_RETRY_INTERVAL = 3  # seconds
DEFAULT_LISTENER_RETRY_COUNT = 5
DEFAULT_LISTENER_INTERRUPT_INTERVAL = 1800  # 30 minutes

R = TypeVar("R")
T = TypeVar("T")
L = TypeVar("L")

SentinelValue = Literal["STOP"]
SENTINEL_VALUE: SentinelValue = "STOP"


TRequest = TypeVar("TRequest")
TResponse = TypeVar("TResponse")


class Subscription(Generic[T]):
    def __init__(self, id: int) -> None:
        self.id = id
        self.queue: asyncio.Queue[T | SentinelValue] = asyncio.Queue()

    async def __aiter__(self) -> "Subscription[T]":
        return self

    async def __anext__(self) -> T | SentinelValue:
        return await self.queue.get()

    async def get(self) -> T:
        event = await self.queue.get()

        if event == "STOP":
            raise StopAsyncIteration

        return event

    async def put(self, item: T) -> None:
        await self.queue.put(item)

    async def close(self) -> None:
        await self.queue.put("STOP")


class PooledListener(Generic[R, T, L], ABC):
    def __init__(self, config: ClientConfig):
        self.token = config.token
        self.config = config

        self.from_subscriptions: dict[int, str] = {}
        self.to_subscriptions: dict[str, list[int]] = {}

        self.subscription_counter: int = 0
        self.subscription_counter_lock: asyncio.Lock = asyncio.Lock()

        self.requests: asyncio.Queue[R | int] = asyncio.Queue()

        self.listener: grpc.aio.UnaryStreamCall[R, T] | None = None
        self.listener_task: asyncio.Task[None] | None = None

        self.curr_requester: int = 0

        self.events: dict[int, Subscription[T]] = {}

        self.interrupter: asyncio.Task[None] | None = None

        ## IMPORTANT: This needs to be created lazily so we don't require
        ## an event loop to instantiate the client.
        self.client: L | None = None

    async def _interrupter(self) -> None:
        """
        _interrupter runs in a separate thread and interrupts the listener according to a configurable duration.
        """
        await asyncio.sleep(DEFAULT_LISTENER_INTERRUPT_INTERVAL)

        if self.interrupt is not None:
            self.interrupt.set()

    @abstractmethod
    def generate_key(self, response: T) -> str:
        pass

    async def _init_producer(self) -> None:
        try:
            if not self.listener:
                while True:
                    try:
                        self.listener = await self._retry_subscribe()

                        logger.debug("Listener connected.")

                        # spawn an interrupter task
                        if self.interrupter is not None and not self.interrupter.done():
                            self.interrupter.cancel()

                        self.interrupter = asyncio.create_task(self._interrupter())

                        while True:
                            self.interrupt = ThreadSafeEvent()
                            if self.listener is None:
                                continue

                            t = asyncio.create_task(
                                read_with_interrupt(
                                    self.listener, self.interrupt, self.generate_key
                                )
                            )
                            await self.interrupt.wait()

                            if not t.done():
                                logger.warning(
                                    "Interrupted read_with_interrupt task of listener"
                                )

                                t.cancel()
                                self.listener.cancel()

                                await asyncio.sleep(DEFAULT_LISTENER_RETRY_INTERVAL)
                                break

                            event = t.result()

                            if isinstance(event, UnexpectedEOF):
                                logger.debug(
                                    f"Handling EOF in Pooled Listener {self.__class__.__name__}"
                                )
                                break

                            subscriptions = self.to_subscriptions.get(event.key, [])

                            for subscription_id in subscriptions:
                                await self.events[subscription_id].put(event.data)

                    except grpc.RpcError as e:
                        logger.debug(f"grpc error in listener: {e}")
                        await asyncio.sleep(DEFAULT_LISTENER_RETRY_INTERVAL)
                        continue

        except Exception as e:
            logger.error(f"Error in listener: {e}")

            self.listener = None

            # close all subscriptions
            for subscription_id in self.events:
                await self.events[subscription_id].close()

            raise e

    @abstractmethod
    def create_request_body(self, item: str) -> R:
        pass

    async def _request(self) -> AsyncIterator[R]:
        self.curr_requester = self.curr_requester + 1

        to_subscribe_to = set(self.from_subscriptions.values())

        for item in to_subscribe_to:
            yield self.create_request_body(item)

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
        id = self.from_subscriptions[subscription_id]

        if id in self.to_subscriptions:
            self.to_subscriptions[id].remove(subscription_id)

        del self.from_subscriptions[subscription_id]
        del self.events[subscription_id]

    async def subscribe(self, id: str) -> T:
        subscription_id: int | None = None

        try:
            async with self.subscription_counter_lock:
                self.subscription_counter += 1
                subscription_id = self.subscription_counter

            self.from_subscriptions[subscription_id] = id

            if id not in self.to_subscriptions:
                self.to_subscriptions[id] = [subscription_id]
            else:
                self.to_subscriptions[id].append(subscription_id)

            self.events[subscription_id] = Subscription(subscription_id)

            await self.requests.put(self.create_request_body(id))

            if not self.listener_task or self.listener_task.done():
                self.listener_task = asyncio.create_task(self._init_producer())

            return await self.events[subscription_id].get()
        except asyncio.CancelledError:
            raise
        finally:
            if subscription_id:
                self.cleanup_subscription(subscription_id)

    @abstractmethod
    async def create_subscription(
        self, request: AsyncIterator[R], metadata: tuple[tuple[str, str]]
    ) -> grpc.aio.UnaryStreamCall[R, T]:
        pass

    async def _retry_subscribe(
        self,
    ) -> grpc.aio.UnaryStreamCall[R, T]:
        retries = 0
        while retries < DEFAULT_LISTENER_RETRY_COUNT:
            try:
                if retries > 0:
                    await asyncio.sleep(DEFAULT_LISTENER_RETRY_INTERVAL)

                # signal previous async iterator to stop
                if self.curr_requester != 0:
                    self.requests.put_nowait(self.curr_requester)

                return await self.create_subscription(
                    self._request(),
                    metadata=get_metadata(self.token),
                )

            except grpc.RpcError as e:  # noqa: PERF203
                if e.code() == grpc.StatusCode.UNAVAILABLE:
                    retries = retries + 1
                else:
                    raise ValueError("gRPC error") from e

        raise ValueError("Failed to connect to listener")
