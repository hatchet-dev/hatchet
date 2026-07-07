from __future__ import annotations

import asyncio
from contextlib import suppress
from typing import Any
from unittest.mock import MagicMock

from hatchet_sdk.worker.runner.runner import Runner

KEY = "step-run-1/0"


def _make_runner() -> Runner:
    runner = Runner.__new__(Runner)
    runner.tasks = {}
    runner.threads = {}
    runner.contexts = {}
    runner.cancellations = {}  # type: ignore[assignment]
    runner.durable_event_listener = None
    runner.durable_eviction_manager = None
    runner.event_queue = MagicMock()
    return runner


def _make_context() -> MagicMock:
    ctx = MagicMock()
    ctx.exit_flag = False
    return ctx


async def _block_forever() -> None:
    await asyncio.Event().wait()


async def _make_running_task() -> asyncio.Task[None]:
    task = asyncio.create_task(_block_forever())
    await asyncio.sleep(0)
    return task


async def _make_cancelled_task() -> asyncio.Task[None]:
    task = await _make_running_task()
    task.cancel()
    with suppress(asyncio.CancelledError):
        await task
    return task


async def _cleanup(*tasks: asyncio.Task[Any]) -> None:
    for task in tasks:
        task.cancel()
        with suppress(asyncio.CancelledError):
            await task


async def test_cleanup_removes_task_and_context() -> None:
    runner = _make_runner()
    task = await _make_running_task()

    runner.tasks[KEY] = task
    runner.contexts[KEY] = _make_context()

    runner.cleanup_run_id(KEY)

    assert KEY not in runner.tasks
    assert KEY not in runner.contexts

    await _cleanup(task)


def _make_action(key: str = KEY) -> MagicMock:
    action = MagicMock()
    action.key = key
    action.step_run_id = "step-run-1"
    action.action_id = "test:action"
    action.retry_count = 0
    return action


async def test_done_callback_consumes_own_cancellation_flag() -> None:
    runner = _make_runner()
    old_task = await _make_cancelled_task()

    runner.cancellations[KEY] = True

    callback = runner.step_run_callback(_make_action(), MagicMock())
    callback(old_task)

    assert KEY not in runner.cancellations
    runner.event_queue.put.assert_not_called()


async def test_done_callbacks_for_different_keys_are_independent() -> None:
    runner = _make_runner()

    key_a = "step-run-1/0/1"
    key_b = "step-run-1/0/2"

    task_a = await _make_cancelled_task()
    task_b = await _make_running_task()
    ctx_b = _make_context()

    runner.tasks[key_b] = task_b
    runner.contexts[key_b] = ctx_b

    callback_a = runner.step_run_callback(_make_action(key_a), MagicMock())
    callback_a(task_a)

    assert runner.tasks.get(key_b) is task_b
    assert runner.contexts.get(key_b) is ctx_b

    await _cleanup(task_b)
