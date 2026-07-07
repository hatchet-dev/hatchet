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


async def test_cleanup_skips_when_key_owned_by_newer_run() -> None:
    runner = _make_runner()
    old_task = await _make_cancelled_task()
    new_task = await _make_running_task()
    new_ctx = _make_context()

    runner.tasks[KEY] = new_task
    runner.contexts[KEY] = new_ctx

    runner.cleanup_run_id(KEY, owner=old_task)

    assert runner.tasks.get(KEY) is new_task
    assert runner.contexts.get(KEY) is new_ctx

    await _cleanup(new_task)


async def test_cleanup_proceeds_for_owner() -> None:
    runner = _make_runner()
    task = await _make_running_task()

    runner.tasks[KEY] = task
    runner.contexts[KEY] = _make_context()

    runner.cleanup_run_id(KEY, owner=task)

    assert KEY not in runner.tasks
    assert KEY not in runner.contexts

    await _cleanup(task)


async def test_cleanup_without_owner_proceeds() -> None:
    runner = _make_runner()
    task = await _make_running_task()

    runner.tasks[KEY] = task
    runner.contexts[KEY] = _make_context()

    runner.cleanup_run_id(KEY)

    assert KEY not in runner.tasks
    assert KEY not in runner.contexts

    await _cleanup(task)


def _make_action() -> MagicMock:
    action = MagicMock()
    action.key = KEY
    action.step_run_id = "step-run-1"
    action.action_id = "test:action"
    action.retry_count = 0
    return action


async def test_stale_done_callback_does_not_consume_new_runs_state() -> None:
    runner = _make_runner()
    old_task = await _make_cancelled_task()
    new_task = await _make_running_task()
    new_ctx = _make_context()

    runner.tasks[KEY] = new_task
    runner.contexts[KEY] = new_ctx
    runner.cancellations[KEY] = True

    callback = runner.step_run_callback(_make_action(), MagicMock())
    callback(old_task)

    assert runner.tasks.get(KEY) is new_task
    assert runner.contexts.get(KEY) is new_ctx
    assert runner.cancellations.get(KEY) is True
    runner.event_queue.put.assert_not_called()

    await _cleanup(new_task)


async def test_done_callback_consumes_own_cancellation_flag() -> None:
    runner = _make_runner()
    old_task = await _make_cancelled_task()

    runner.cancellations[KEY] = True

    callback = runner.step_run_callback(_make_action(), MagicMock())
    callback(old_task)

    assert KEY not in runner.cancellations
    runner.event_queue.put.assert_not_called()
