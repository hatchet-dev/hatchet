"""Regression tests for issue #4803: internal executor isolation.

These tests verify that Hatchet SDK internal blocking operations
can proceed even when the event-loop's **default** executor pool
is fully saturated by user code, and that contextvars are preserved.

The tests do NOT require a live Hatchet backend, Go engine, Docker,
network, or API token.
"""

from __future__ import annotations

import asyncio
import contextvars
import threading
from concurrent.futures import ThreadPoolExecutor
from typing import Any

import pytest


_TINY_POOL_SIZE = 2


def _block_for_executor(released: threading.Event) -> None:
    """Block until *released* is set (or 30 s timeout)."""
    released.wait(timeout=30)


class TestDefaultExecutorStarvation:
    """Prove that asyncio.to_thread is blocked when the default executor
    is saturated (the bug reported in #4803).
    """

    async def test_asyncio_to_thread_blocked_by_saturated_default_executor(
        self,
    ) -> None:
        loop = asyncio.get_running_loop()
        released = threading.Event()

        tiny_executor = ThreadPoolExecutor(max_workers=_TINY_POOL_SIZE)
        loop.set_default_executor(tiny_executor)

        try:
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            await asyncio.sleep(0.05)

            done = threading.Event()

            async def _to_thread_call() -> None:
                await asyncio.to_thread(done.wait, timeout=0.5)

            with pytest.raises(asyncio.TimeoutError):
                await asyncio.wait_for(_to_thread_call(), timeout=2.0)

            assert not done.is_set()

        finally:
            released.set()
            tiny_executor.shutdown(wait=False, cancel_futures=True)


class TestHatchetInternalExecutorIsolation:
    """Verify that hatchet_to_thread completes even when the default
    executor is saturated — this is the fix for #4803.
    """

    async def test_hatchet_to_thread_completes_with_saturated_default_executor(
        self,
    ) -> None:
        from hatchet_sdk.utils.executor import hatchet_to_thread

        loop = asyncio.get_running_loop()
        released = threading.Event()

        tiny_executor = ThreadPoolExecutor(max_workers=_TINY_POOL_SIZE)
        loop.set_default_executor(tiny_executor)

        try:
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            await asyncio.sleep(0.05)

            result = await hatchet_to_thread(lambda: 42)
            assert result == 42

        finally:
            released.set()
            tiny_executor.shutdown(wait=False, cancel_futures=True)

    async def test_internal_work_not_blocked_by_user_thread_calls(
        self,
    ) -> None:
        from hatchet_sdk.utils.executor import hatchet_to_thread

        loop = asyncio.get_running_loop()
        released = threading.Event()

        tiny_executor = ThreadPoolExecutor(max_workers=_TINY_POOL_SIZE)
        loop.set_default_executor(tiny_executor)

        try:
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            loop.run_in_executor(tiny_executor, _block_for_executor, released)
            await asyncio.sleep(0.05)

            results: list[int] = []

            async def _internal_work(value: int) -> None:
                res = await hatchet_to_thread(lambda v=value: v * 10)
                results.append(res)

            await asyncio.gather(*[_internal_work(i) for i in range(5)])

            assert sorted(results) == [0, 10, 20, 30, 40]

        finally:
            released.set()
            tiny_executor.shutdown(wait=False, cancel_futures=True)

    async def test_hatchet_to_thread_with_kwargs(self) -> None:
        from hatchet_sdk.utils.executor import hatchet_to_thread

        def _add(a: int, b: int) -> int:
            return a + b

        result = await hatchet_to_thread(_add, 3, b=7)
        assert result == 10

    async def test_hatchet_to_thread_exception_propagation(self) -> None:
        from hatchet_sdk.utils.executor import hatchet_to_thread

        def _fail() -> None:
            raise ValueError("boom")

        with pytest.raises(ValueError, match="boom"):
            await hatchet_to_thread(_fail)

    async def test_contextvars_propagation(self) -> None:
        """contextvars must propagate through hatchet_to_thread identically
        to asyncio.to_thread — this is critical for Hatchet SDK internals
        that rely on contextvars (ctx_step_run_id, ctx_worker_id, etc.).
        """
        from hatchet_sdk.utils.executor import hatchet_to_thread

        test_var = contextvars.ContextVar("test_var", default="unset")
        test_var.set("hello_from_caller")

        def _read_in_thread() -> str:
            return test_var.get()

        # hatchet_to_thread must see the value set in the calling context
        result = await hatchet_to_thread(_read_in_thread)
        assert result == "hello_from_caller"

        # Verify isolation: the value should NOT leak into a new context
        test_var.set("second_value")

        async def _read_in_new_ctx() -> str:
            # Run in a fresh context (simulating a new task)
            ctx = contextvars.copy_context()
            loop = asyncio.get_running_loop()
            from hatchet_sdk.utils.executor import _get_executor

            def _inner() -> str:
                return test_var.get()

            return await loop.run_in_executor(_get_executor(), ctx.run, _inner)

        # New context should have the copied value
        result2 = await _read_in_new_ctx()
        assert result2 == "second_value"


class TestShutdownExecutor:
    """Verify executor lifecycle."""

    def test_shutdown_executor_is_idempotent(self) -> None:
        from hatchet_sdk.utils.executor import shutdown_executor

        shutdown_executor()
        shutdown_executor()

    def test_executor_lazy_creation(self) -> None:
        import hatchet_sdk.utils.executor as mod

        mod._executor = None

        assert mod._executor is None

        executor = mod._get_executor()
        assert executor is not None
        assert mod._executor is executor

        mod.shutdown_executor()
