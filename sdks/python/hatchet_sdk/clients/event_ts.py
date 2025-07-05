import asyncio
from collections.abc import Callable
from typing import Generic, TypeVar, cast, overload

import grpc.aio
from grpc._cython import cygrpc  # type: ignore[attr-defined]

from hatchet_sdk.logger import logger


class ThreadSafeEvent(asyncio.Event):
    """
    ThreadSafeEvent is a subclass of asyncio.Event that allows for thread-safe setting and clearing of the event.
    """

    def __init__(self) -> None:
        super().__init__()
        if self._loop is None:  # type: ignore[has-type]
            self._loop = asyncio.get_event_loop()

    def set(self) -> None:
        if not self._loop.is_closed():
            self._loop.call_soon_threadsafe(super().set)

    def clear(self) -> None:
        self._loop.call_soon_threadsafe(super().clear)


TRequest = TypeVar("TRequest")
TResponse = TypeVar("TResponse")


class ReadWithInterruptResult(Generic[TResponse]):
    def __init__(self, data: TResponse, key: str):
        self.data = data
        self.key = key


class UnexpectedEOF:
    def __init__(self) -> None:
        pass


@overload
async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: Callable[[TResponse], str],
) -> ReadWithInterruptResult[TResponse] | UnexpectedEOF: ...


@overload
async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: None = None,
) -> ReadWithInterruptResult[TResponse] | UnexpectedEOF: ...


async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: Callable[[TResponse], str] | None = None,
) -> ReadWithInterruptResult[TResponse] | UnexpectedEOF:
    try:
        result = cast(TResponse, await listener.read())

        if result is cygrpc.EOF:
            logger.warning("Received EOF from engine")
            return UnexpectedEOF()

        key = key_generator(result) if key_generator else ""

        return ReadWithInterruptResult(data=result, key=key)
    finally:
        interrupt.set()
