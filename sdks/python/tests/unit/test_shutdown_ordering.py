"""Tests for worker shutdown ordering and subprocess coordination."""

from __future__ import annotations

import asyncio
import multiprocessing
import threading
import time
from dataclasses import dataclass
from typing import Any
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from hatchet_sdk.clients.dispatcher.action_listener import ActionListener
from hatchet_sdk.runnables.action import ActionType
from hatchet_sdk.utils.typing import STOP_LOOP
from hatchet_sdk.worker.action_listener_process import (
    ActionEvent,
    WorkerActionListenerProcess,
    worker_action_listener_process,
)

_LISTENER_MODULE = "hatchet_sdk.worker.action_listener_process"
_ACTION_LISTENER_MODULE = "hatchet_sdk.clients.dispatcher.action_listener"

_CTX = multiprocessing.get_context("spawn")


@dataclass
class _FakeAction:
    action_type: Any = ActionType.START_STEP_RUN
    action_id: str = "test-action-id"


def _make_listener(worker_id: str = "test-worker-id") -> ActionListener:
    with patch(f"{_ACTION_LISTENER_MODULE}.new_conn"):
        listener = ActionListener(config=MagicMock(), worker_id=worker_id)
    listener.cleanup = MagicMock()  # type: ignore[method-assign]
    listener.get_listen_client = AsyncMock(  # type: ignore[method-assign]
        side_effect=Exception("no gRPC server in tests")
    )
    return listener


def _make_process(
    stop_event: Any,
    worker_id_queue: Any,
    event_queue: Any = None,
) -> WorkerActionListenerProcess:
    config = MagicMock()
    config.healthcheck.enabled = False

    if event_queue is None:
        event_queue = _CTX.Queue()

    with patch(f"{_LISTENER_MODULE}.Client"):
        process = WorkerActionListenerProcess(
            name="test-worker",
            actions=["test_action"],
            slot_config={"default": 1, "durable": 0},
            config=config,
            action_queue=_CTX.Queue(),
            event_queue=event_queue,
            handle_kill=False,
            debug=False,
            labels=[],
            worker_id_queue=worker_id_queue,
            stop_event=stop_event,
        )
    return process


async def _cancel_tasks(*tasks: asyncio.Task[Any] | None) -> None:
    for task in tasks:
        if task is not None and not task.done():
            task.cancel()
            with pytest.raises((asyncio.CancelledError, Exception)):
                await task


async def test_stop_event_stops_action_loop_without_kill() -> None:
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()

    process = _make_process(stop_event, worker_id_queue)
    listener = _make_listener()
    process.listener = listener

    task = asyncio.create_task(process._wait_for_stop_event())

    # Let the task start and block in its executor thread.
    await asyncio.sleep(0.05)
    assert not process.killing, "should not be killing before event is set"

    # Trigger shutdown via the event — the primary (non-signal) path.
    stop_event.set()
    await asyncio.wait_for(task, timeout=5.0)

    assert process.killing, "_stop_action_loop should have set killing=True"
    listener.cleanup.assert_called_once()


async def test_stop_action_loop_is_idempotent() -> None:
    """_stop_action_loop must not raise or double-call cleanup when called twice."""
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()

    process = _make_process(stop_event, worker_id_queue)
    listener = _make_listener()
    process.listener = listener

    await process._stop_action_loop()
    assert process.killing

    # Second call must succeed silently.
    await process._stop_action_loop()
    # cleanup() is called exactly once (the second call short-circuits).
    listener.cleanup.assert_called_once()  # type: ignore[attr-defined]


async def test_worker_id_published_to_queue_on_start() -> None:
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()
    event_queue: Any = _CTX.Queue()

    listener = _make_listener()

    mock_dispatcher = AsyncMock()
    mock_dispatcher.get_action_listener = AsyncMock(return_value=listener)

    process = _make_process(stop_event, worker_id_queue, event_queue)

    with (
        patch(f"{_LISTENER_MODULE}.DispatcherClient", return_value=mock_dispatcher),
        patch(f"{_LISTENER_MODULE}.Client"),
    ):
        await process.start()

    try:
        # worker_id must be enqueued shortly after start().
        # Use get(timeout) rather than empty()+get_nowait() to avoid the
        # multiprocessing.Queue feeder-thread race where put() has been called
        # but the item hasn't been flushed to the pipe yet.
        try:
            published_id = worker_id_queue.get(timeout=1.0)
        except Exception:
            pytest.fail("worker_id was not enqueued within 1s of start()")
        assert published_id == "test-worker-id"
    finally:
        event_queue.put(STOP_LOOP)  # exits event_send_loop
        stop_event.set()  # exits _wait_for_stop_event

        await asyncio.sleep(0.1)
        all_tasks = [
            process.action_loop_task,
            process.event_send_loop_task,
            process.blocked_main_loop,
            process._stop_event_task,
        ]
        remaining = [t for t in all_tasks if t is not None and not t.done()]
        if remaining:
            for t in remaining:
                t.cancel()
            await asyncio.gather(*remaining, return_exceptions=True)


def test_event_queue_drains_before_process_exits() -> None:
    """All events put into event_queue must be consumed before the subprocess exits.

    Runs worker_action_listener_process in a thread (it calls asyncio.run()
    internally) so we can exercise the full function with real queues without
    a costly multiprocessing spawn.
    """
    action_queue: Any = _CTX.Queue()
    event_queue: Any = _CTX.Queue()
    worker_id_queue: Any = _CTX.Queue()
    stop_event = _CTX.Event()

    config = MagicMock()
    config.healthcheck.enabled = False

    consumed_events: list[str] = []
    stub_listener = _make_listener()

    async def fake_send(
        action: Any, ev_type: Any, payload: Any, should_not_retry: Any
    ) -> None:
        consumed_events.append(str(ev_type))

    mock_dispatcher = AsyncMock()
    mock_dispatcher.get_action_listener = AsyncMock(return_value=stub_listener)
    mock_dispatcher.send_step_action_event = AsyncMock(side_effect=fake_send)

    def run_subprocess() -> None:
        with (
            patch(f"{_LISTENER_MODULE}.DispatcherClient", return_value=mock_dispatcher),
            patch(f"{_LISTENER_MODULE}.Client"),
        ):
            worker_action_listener_process(
                name="test-worker",
                actions=["test_action"],
                slot_config={"default": 1, "durable": 0},
                config=config,
                action_queue=action_queue,
                event_queue=event_queue,
                handle_kill=False,
                debug=False,
                labels=[],
                worker_id_queue=worker_id_queue,
                stop_event=stop_event,
            )

    thread = threading.Thread(target=run_subprocess, daemon=True)
    thread.start()

    # Wait for the subprocess to publish its worker_id (ready signal).
    deadline = time.monotonic() + 10.0
    while worker_id_queue.empty():
        assert time.monotonic() < deadline, "subprocess never published worker_id"
        time.sleep(0.05)

    # Simulate the runner finishing a task: put a completion event in the queue.
    # _FakeAction is a module-level dataclass so it can be pickled across the
    # multiprocessing.Queue feeder thread.
    fake_event = ActionEvent(
        action=_FakeAction(),  # type: ignore
        type=ActionType.START_STEP_RUN,
        payload=None,
        should_not_retry=False,
    )
    event_queue.put(fake_event)

    # Allow the event_send_loop a moment to pick up the event.
    time.sleep(0.1)

    stop_event.set()
    time.sleep(0.05)
    event_queue.put(STOP_LOOP)  # stops event_send_loop

    thread.join(timeout=15.0)
    assert not thread.is_alive(), "subprocess thread should have exited after STOP_LOOP"

    # The event_queue must be fully drained before the subprocess exited.
    assert (
        event_queue.empty()
    ), "event_queue must be fully drained before subprocess exits"
