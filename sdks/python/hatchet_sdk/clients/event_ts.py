import asyncio
from typing import Callable, TypeVar, cast, overload

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


@overload
async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: Callable[[TResponse], str],
) -> tuple[TResponse, str, bool]: ...


@overload
async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: None = None,
) -> tuple[TResponse, None, bool]: ...


async def read_with_interrupt(
    listener: grpc.aio.UnaryStreamCall[TRequest, TResponse],
    interrupt: ThreadSafeEvent,
    key_generator: Callable[[TResponse], str] | None = None,
) -> tuple[TResponse, str | None, bool]:
    try:
        result = cast(TResponse, await listener.read())

        if result is cygrpc.EOF:
            logger.warning("Received EOF from engine")
            return cast(TResponse, None), None, True

        key = key_generator(result) if key_generator else None

        return result, key, False
    finally:
        interrupt.set()
