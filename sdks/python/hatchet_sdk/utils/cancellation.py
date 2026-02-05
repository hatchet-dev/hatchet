"""Utilities for cancellation-aware operations."""

from __future__ import annotations

import asyncio
import contextlib
import inspect
from collections.abc import Awaitable, Callable
from typing import TYPE_CHECKING, TypeVar

from hatchet_sdk.logger import logger

if TYPE_CHECKING:
    from hatchet_sdk.cancellation import CancellationToken

T = TypeVar("T")


async def await_with_cancellation(
    coro: Awaitable[T],
    token: CancellationToken | None,
    cancel_callback: Callable[[], Awaitable[None] | None] | None = None,
) -> T:
    """
    Await an awaitable with cancellation support.

    This function races the given awaitable against a cancellation token. If the
    token is cancelled before the awaitable completes, the awaitable is cancelled
    and an asyncio.CancelledError is raised.

    Args:
        coro: The awaitable to await (coroutine, Future, or Task).
        token: The cancellation token to check. If None, the coroutine is awaited directly.
        cancel_callback: An optional callback to invoke when cancellation occurs
            (e.g., to cancel child workflows). May be sync or async.

    Returns:
        The result of the coroutine.

    Raises:
        asyncio.CancelledError: If the token is cancelled before the coroutine completes.

    Example:
        ```python
        def cleanup() -> None:
            print("cleaning up...")

        async def long_running_task():
            await asyncio.sleep(10)
            return "done"

        token = CancellationToken()

        # This will raise asyncio.CancelledError if token.cancel() is called
        result = await await_with_cancellation(
            long_running_task(),
            token,
            cancel_callback=cleanup,
        )
        ```
    """

    async def _invoke_cancel_callback() -> None:
        if not cancel_callback:
            return

        result = cancel_callback()
        if inspect.isawaitable(result):
            await result

    if token is None:
        logger.debug("await_with_cancellation: no token provided, awaiting directly")
        return await coro

    logger.debug("await_with_cancellation: starting with cancellation token")

    # Check if already cancelled
    if token.is_cancelled:
        logger.debug("await_with_cancellation: token already cancelled")
        if cancel_callback:
            logger.debug("await_with_cancellation: invoking cancel callback")
            await _invoke_cancel_callback()
        raise asyncio.CancelledError("Operation cancelled by cancellation token")

    main_task = asyncio.ensure_future(coro)
    cancel_task = asyncio.create_task(token.aio_wait())

    try:
        done, pending = await asyncio.wait(
            [main_task, cancel_task],
            return_when=asyncio.FIRST_COMPLETED,
        )

        # Cancel pending tasks
        for task in pending:
            task.cancel()
            with contextlib.suppress(asyncio.CancelledError):
                await task

        if cancel_task in done:
            logger.debug("await_with_cancellation: cancelled before completion")
            if cancel_callback:
                logger.debug("await_with_cancellation: invoking cancel callback")
                await _invoke_cancel_callback()
            raise asyncio.CancelledError("Operation cancelled by cancellation token")

        logger.debug("await_with_cancellation: completed successfully")
        return main_task.result()

    except asyncio.CancelledError:
        # If we're cancelled externally (not via token), also invoke callback
        logger.debug("await_with_cancellation: externally cancelled")
        if cancel_callback:
            logger.debug("await_with_cancellation: invoking cancel callback")
            with contextlib.suppress(asyncio.CancelledError):
                await asyncio.shield(_invoke_cancel_callback())
        main_task.cancel()
        cancel_task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await main_task
        with contextlib.suppress(asyncio.CancelledError):
            await cancel_task
        raise
