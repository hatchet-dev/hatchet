"""Regression tests for graceful shutdown when the engine "pause task
assignment" REST call misbehaves.

`Worker.exit_gracefully()` calls `_pause_task_assignment()`, which is a
best-effort courtesy to the engine (so it stops routing new work to a worker
that is draining). It must never be a precondition for the worker's own
local teardown: if the REST call raises, times out, or simply never
returns, `exit_gracefully()` must still reach runner drain, listener/queue
teardown, and loop stop.
"""

from __future__ import annotations

import asyncio
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from hatchet_sdk.utils.typing import STOP_LOOP
from hatchet_sdk.worker import worker as worker_module
from hatchet_sdk.worker.worker import Worker

pytestmark = pytest.mark.asyncio


def _make_worker(aio_pause: AsyncMock) -> Worker:
    """Build a Worker with just enough state for exit_gracefully() to run,
    bypassing __init__ (which spins up multiprocessing queues, a real
    Client, and process-wide signal handlers we don't want in a unit test).
    """
    w = Worker.__new__(Worker)

    w._killing = False
    w._name = "test-worker"

    # worker_id_queue.get(timeout=...) is called via asyncio.to_thread, so a
    # plain Mock with a synchronous get() is enough.
    worker_id_queue = MagicMock()
    worker_id_queue.get = MagicMock(return_value="test-worker-id")
    w._worker_id_queue = worker_id_queue

    client = MagicMock()
    client.workers.aio_pause = aio_pause
    w._client = client

    w._action_runner = MagicMock()
    w._action_runner.exit_gracefully = AsyncMock()
    w._legacy_durable_action_runner = None

    w._stop_listener_event = MagicMock()
    w._event_queue = MagicMock()
    w._durable_event_queue = None
    w._durable_action_listener_process = None

    w._action_queue = MagicMock()
    w._durable_action_queue = None

    w._lifespan_stack = None
    w._lifespan_cleanup_complete = None

    # An already-resolved Future stands in for the health-check task that
    # `_close()` awaits.
    health_check: asyncio.Future[None] = asyncio.get_event_loop().create_future()
    health_check.set_result(None)
    w._action_listener_health_check = health_check  # type: ignore[assignment]

    # Mocked loop so exit_gracefully()'s final loop.stop() doesn't tear down
    # the loop that's running the test itself.
    w._loop = MagicMock()

    return w


async def _run_exit_gracefully(w: Worker) -> None:
    # Generous timeout: if this fires, exit_gracefully() is hanging, which is
    # exactly the regression this test guards against.
    await asyncio.wait_for(w.exit_gracefully(), timeout=5.0)


async def test_exit_gracefully_completes_when_engine_pause_raises() -> None:
    """A REST error (503, connection error, engine roll, etc.) from
    aio_pause() must not stop local teardown from proceeding."""
    aio_pause = AsyncMock(side_effect=RuntimeError("engine unreachable"))
    w = _make_worker(aio_pause)

    await _run_exit_gracefully(w)

    aio_pause.assert_awaited_once_with("test-worker-id")

    assert w._killing is True
    w._action_runner.exit_gracefully.assert_awaited_once()
    w._stop_listener_event.set.assert_called_once()
    w._event_queue.put.assert_called_once_with(STOP_LOOP)
    w._loop.stop.assert_called_once()


async def test_exit_gracefully_completes_when_engine_pause_never_returns() -> None:
    """A hung REST call (DNS failure, dropped connection, engine that never
    responds) must be bounded by a timeout rather than blocking shutdown
    forever."""

    async def _hang(*_args: Any, **_kwargs: Any) -> None:
        await asyncio.Event().wait()  # never completes

    aio_pause = AsyncMock(side_effect=_hang)
    w = _make_worker(aio_pause)

    # Use a short timeout so the test doesn't have to wait out the real
    # (production) timeout to prove the behavior.
    with patch.object(worker_module, "_PAUSE_TASK_ASSIGNMENT_TIMEOUT_SECONDS", 0.05):
        await _run_exit_gracefully(w)

    assert w._killing is True
    w._action_runner.exit_gracefully.assert_awaited_once()
    w._stop_listener_event.set.assert_called_once()
    w._event_queue.put.assert_called_once_with(STOP_LOOP)
    w._loop.stop.assert_called_once()


async def test_pause_task_assignment_swallows_errors_directly() -> None:
    """_pause_task_assignment() itself must never raise, independent of
    exit_gracefully()'s own guard, since other callers may invoke it."""
    aio_pause = AsyncMock(side_effect=RuntimeError("boom"))
    w = _make_worker(aio_pause)

    await asyncio.wait_for(w._pause_task_assignment(), timeout=5.0)

    aio_pause.assert_awaited_once_with("test-worker-id")


async def test_signal_handler_exceptions_are_logged_not_dropped() -> None:
    """The SIGTERM/SIGINT handlers schedule exit_gracefully() with
    create_task(); if that task raises, the exception must be observed and
    logged rather than silently vanishing."""
    w = Worker.__new__(Worker)
    w._loop = asyncio.get_event_loop()

    async def _boom() -> None:
        raise RuntimeError("shutdown blew up")

    with patch.object(worker_module.logger, "error") as mock_log_error:
        w._create_tracked_task(_boom(), "exit_gracefully")
        # Let the scheduled task run and its done-callback fire.
        await asyncio.sleep(0.05)

    mock_log_error.assert_called_once()
    args, kwargs = mock_log_error.call_args
    assert "exit_gracefully" in args[0]
    assert isinstance(kwargs.get("exc_info"), RuntimeError)
