"""Dedicated internal executor for Hatchet SDK control-plane blocking work.

This module provides a private, lifecycle-managed ``ThreadPoolExecutor`` used
by the SDK's own ``asyncio.to_thread`` calls (e.g. logging, label updates,
timeout refreshes, dependency resolution).

The goal is to isolate Hatchet-internal blocking work from user code that
may call ``asyncio.to_thread`` or ``loop.run_in_executor(None, ...)``.
User code shares the event-loop **default** executor; this module gives the
SDK its own pool so user saturation of the default executor cannot starve
SDK control-plane operations (issue #4803).

Lifecycle
---------
* The executor is created lazily on first use.
* ``shutdown_executor()`` must be called during worker shutdown to avoid
  thread leaks.  It is safe to call multiple times.
"""

from __future__ import annotations

import asyncio
import contextvars
import functools
import os
from concurrent.futures import ThreadPoolExecutor
from typing import Any, TypeVar

_T = TypeVar("_T")

_executor: ThreadPoolExecutor[Any] | None = None

# Match CPython's default: min(32, os.cpu_count() + 4)
_DEFAULT_WORKERS = min(32, (os.cpu_count() or 1) + 4)


def _get_executor() -> ThreadPoolExecutor[Any]:
    """Return (and lazily create) the dedicated Hatchet internal executor."""
    global _executor
    if _executor is None:
        _executor = ThreadPoolExecutor(max_workers=_DEFAULT_WORKERS)
    return _executor


async def hatchet_to_thread(fn: Any, /, *args: Any, **kwargs: Any) -> Any:
    """Run *fn* on the dedicated Hatchet internal executor.

    Drop-in replacement for ``asyncio.to_thread`` that avoids the default
    executor pool.  Accepts both positional and keyword arguments.

    Preserves ``contextvars`` context identically to ``asyncio.to_thread``:
    a snapshot of the current context is captured at call time and replayed
    in the worker thread.
    """
    loop = asyncio.get_running_loop()
    ctx = contextvars.copy_context()

    if kwargs:
        fn = functools.partial(fn, *args, **kwargs)
        args = ()

    return await loop.run_in_executor(_get_executor(), ctx.run, fn, *args)


def shutdown_executor(*, wait: bool = True, cancel_futures: bool = False) -> None:
    """Shut down the internal executor.

    Intended to be called once during worker shutdown.
    Safe to call multiple times (no-op if already shut down).
    """
    global _executor
    if _executor is not None:
        _executor.shutdown(wait=wait, cancel_futures=cancel_futures)
        _executor = None
