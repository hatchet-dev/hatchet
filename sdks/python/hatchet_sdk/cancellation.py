"""Cancellation token for coordinating cancellation across async and sync operations."""

from __future__ import annotations

import asyncio
import threading
from collections.abc import Callable

from hatchet_sdk.exceptions import CancellationReason
from hatchet_sdk.logger import logger


class CancellationToken:
    """
    A token that can be used to signal cancellation across async and sync operations.

    The token provides both asyncio and threading event primitives, allowing it to work
    seamlessly in both async and sync code paths. Child workflow run IDs can be registered
    with the token so they can be cancelled when the parent is cancelled.

    Example:
        ```python
        token = CancellationToken()

        # In async code
        await token.aio_wait()  # Blocks until cancelled

        # In sync code
        token.wait(timeout=1.0)  # Returns True if cancelled within timeout

        # Check if cancelled
        if token.is_cancelled:
            raise CancelledError("Operation was cancelled")

        # Trigger cancellation
        token.cancel()
        ```
    """

    def __init__(self) -> None:
        self._cancelled = False
        self._reason: CancellationReason | None = None
        self._async_event: asyncio.Event | None = None
        self._sync_event = threading.Event()
        self._child_run_ids: list[str] = []
        self._callbacks: list[Callable[[], None]] = []
        self._lock = threading.Lock()

    def _get_async_event(self) -> asyncio.Event:
        """Lazily create the asyncio event to avoid requiring an event loop at init time."""
        if self._async_event is None:
            self._async_event = asyncio.Event()
            # If already cancelled, set the event
            if self._cancelled:
                self._async_event.set()
        return self._async_event

    def cancel(
        self, reason: CancellationReason = CancellationReason.TOKEN_CANCELLED
    ) -> None:
        """
        Trigger cancellation.

        This will:
        - Set the cancelled flag and reason
        - Signal both async and sync events
        - Invoke all registered callbacks

        Args:
            reason: The reason for cancellation.
        """
        with self._lock:
            if self._cancelled:
                return

            self._cancelled = True
            self._reason = reason

            # Signal both event types
            if self._async_event is not None:
                self._async_event.set()
            self._sync_event.set()

            # Snapshot callbacks under the lock, invoke outside to avoid deadlocks
            callbacks = list(self._callbacks)

        for callback in callbacks:
            try:
                callback()
            except Exception as e:  # noqa: PERF203
                logger.warning(f"CancellationToken: callback raised exception: {e}")

    @property
    def is_cancelled(self) -> bool:
        """Check if cancellation has been triggered."""
        return self._cancelled

    @property
    def reason(self) -> CancellationReason | None:
        """Get the reason for cancellation, or None if not cancelled."""
        return self._reason

    async def aio_wait(self) -> None:
        """
        Await until cancelled (for use in asyncio).

        This will block until cancel() is called.
        """
        await self._get_async_event().wait()

    def wait(self, timeout: float | None = None) -> bool:
        """
        Block until cancelled (for use in sync code).

        Args:
            timeout: Maximum time to wait in seconds. None means wait forever.

        Returns:
            True if the token was cancelled (event was set), False if timeout expired.
        """
        return self._sync_event.wait(timeout)

    def register_child(self, run_id: str) -> None:
        """
        Register a child workflow run ID with this token.

        When the parent is cancelled, these child run IDs can be used to cancel
        the child workflows as well.

        Args:
            run_id: The workflow run ID of the child workflow.
        """
        with self._lock:
            self._child_run_ids.append(run_id)

    @property
    def child_run_ids(self) -> list[str]:
        """The registered child workflow run IDs."""
        return self._child_run_ids

    def add_callback(self, callback: Callable[[], None]) -> None:
        """
        Register a callback to be invoked when cancellation is triggered.

        If the token is already cancelled, the callback will be invoked immediately.

        Args:
            callback: A callable that takes no arguments.
        """
        with self._lock:
            if self._cancelled:
                invoke_now = True
            else:
                invoke_now = False
                self._callbacks.append(callback)

        if invoke_now:
            try:
                callback()
            except Exception as e:
                logger.warning(f"CancellationToken: callback raised exception: {e}")

    def __repr__(self) -> str:
        return (
            f"CancellationToken(cancelled={self._cancelled}, "
            f"children={len(self._child_run_ids)}, callbacks={len(self._callbacks)})"
        )
