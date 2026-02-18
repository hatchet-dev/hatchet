from __future__ import annotations

import asyncio
import contextlib
from collections.abc import Awaitable, Callable
from typing import TypeVar

from hatchet_sdk.cancellation import CancellationToken
from hatchet_sdk.logger import logger

T = TypeVar("T")


async def _invoke_cancel_callback(
    cancel_callback: Callable[[], Awaitable[None]] | None,
) -> None:
    if not cancel_callback:
        return

    await cancel_callback()


async def race_against_token(
    main_task: asyncio.Task[T],
    token: CancellationToken,
) -> T:
    """
    Race an asyncio task against a cancellation token.

    Waits for either the task to complete or the token to be cancelled. Cleans up
    whichever side loses the race.

    :param main_task: The asyncio task to race.
    :param token: The cancellation token to race against.

    :returns: The result of the main task if it completes first.

    :raises: asyncio.CancelledError: If the token fires before the task completes.
    """
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
            raise asyncio.CancelledError("Operation cancelled by cancellation token")

        return main_task.result()

    except asyncio.CancelledError:
        # Ensure both tasks are cleaned up on any cancellation (external or token)
        main_task.cancel()
        cancel_task.cancel()
        with contextlib.suppress(asyncio.CancelledError):
            await main_task
        with contextlib.suppress(asyncio.CancelledError):
            await cancel_task
        raise


async def await_with_cancellation(
    coro: Awaitable[T],
    token: CancellationToken | None,
    cancel_callback: Callable[[], Awaitable[None]] | None = None,
) -> T:
    """
    Await an awaitable with cancellation support.

    This function races the given awaitable against a cancellation token. If the
    token is cancelled before the awaitable completes, the awaitable is cancelled
    and an asyncio.CancelledError is raised.

    :param coro: The awaitable to await (coroutine, Future, or asyncio.Task).
    :param token: The cancellation token to check. If None, the coroutine is awaited directly.
    :param cancel_callback: An optional async callback to invoke when cancellation occurs (e.g., to cancel child workflows).

    :returns: The result of the coroutine.

    :raises: asyncio.CancelledError: If the token is cancelled before the coroutine completes.

    **Example**

    ```python
    async def cleanup() -> None:
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

    if token is None:
        return await coro

    if token.is_cancelled:
        logger.debug("await_with_cancellation: token already cancelled")
        if cancel_callback:
            logger.debug("await_with_cancellation: invoking cancel callback")
            await _invoke_cancel_callback(cancel_callback)

        raise asyncio.CancelledError("Operation cancelled by cancellation token")

    main_task = asyncio.ensure_future(coro)

    try:
        result = await race_against_token(main_task, token)
        logger.debug("await_with_cancellation: completed successfully")

        return result
    except asyncio.CancelledError:
        if cancel_callback:
            with contextlib.suppress(asyncio.CancelledError):
                await asyncio.shield(_invoke_cancel_callback(cancel_callback))

        raise
