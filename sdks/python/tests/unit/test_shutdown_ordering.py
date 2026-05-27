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

from hatchet_sdk.utils.typing import STOP_LOOP
from hatchet_sdk.worker.action_listener_process import (
    ActionEvent,
    WorkerActionListenerProcess,
    worker_action_listener_process,
)

_LISTENER_MODULE = "hatchet_sdk.worker.action_listener_process"

_CTX = multiprocessing.get_context("spawn")


@dataclass
class _FakeAction:
    """Picklable stand-in for an Action used in event-queue drain tests."""

    action_type: Any = None
    action_id: str = "test-action-id"


class _StubActionListener:
    """Minimal stand-in for an ActionListener gRPC stream.

    Blocks in __aiter__ until stop_signal is set, so tests control exactly
    when the action loop exits.
    """

    def __init__(self) -> None:
        self.stop_signal: bool = False
        self.worker_id: str = "test-worker-id"
        self.cleanup_called: bool = False

    def cleanup(self) -> None:
        self.cleanup_called = True

    def __aiter__(self) -> "_StubActionListener":
        return self

    async def __anext__(self) -> None:
        while not self.stop_signal:
            await asyncio.sleep(0.01)
        raise StopAsyncIteration


def _make_process(
    stop_event: Any,
    worker_id_queue: Any,
    event_queue: Any = None,
) -> WorkerActionListenerProcess:
    """Create a WorkerActionListenerProcess with minimal mocking."""
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
    """Cancel asyncio tasks and wait for them to finish."""
    for task in tasks:
        if task is not None and not task.done():
            task.cancel()
            with pytest.raises((asyncio.CancelledError, Exception)):
                await task


# ---------------------------------------------------------------------------
# Test 1: stop_event fires _stop_action_loop() without any os.kill
# ---------------------------------------------------------------------------


async def test_stop_event_stops_action_loop_without_kill() -> None:
    """Setting the stop_event triggers _stop_action_loop — no os.kill needed."""
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()

    process = _make_process(stop_event, worker_id_queue)
    process.listener = _StubActionListener()  # type: ignore[assignment]

    task = asyncio.create_task(process._wait_for_stop_event())

    # Let the task start and block in its executor thread.
    await asyncio.sleep(0.05)
    assert not process.killing, "should not be killing before event is set"

    # Trigger shutdown via the event — the primary (non-signal) path.
    stop_event.set()
    await asyncio.wait_for(task, timeout=5.0)

    assert process.killing, "_stop_action_loop should have set killing=True"
    assert process.listener.cleanup_called, "listener.cleanup() should have been called"


# ---------------------------------------------------------------------------
# Test 2: _stop_action_loop is idempotent (safe to call twice)
# ---------------------------------------------------------------------------


async def test_stop_action_loop_is_idempotent() -> None:
    """_stop_action_loop must not raise or double-call cleanup when called twice."""
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()

    process = _make_process(stop_event, worker_id_queue)
    stub = _StubActionListener()
    process.listener = stub  # type: ignore[assignment]

    await process._stop_action_loop()
    assert process.killing

    # Second call must succeed silently.
    await process._stop_action_loop()
    # cleanup() is called exactly once (the second call short-circuits).
    assert stub.cleanup_called


# ---------------------------------------------------------------------------
# Test 3: worker_id is published to worker_id_queue during start()
# ---------------------------------------------------------------------------


async def test_worker_id_published_to_queue_on_start() -> None:
    """The listener must put its worker_id in worker_id_queue at startup."""
    stop_event = _CTX.Event()
    worker_id_queue: Any = _CTX.Queue()
    event_queue: Any = _CTX.Queue()

    stub = _StubActionListener()

    mock_dispatcher = AsyncMock()
    mock_dispatcher.get_action_listener = AsyncMock(return_value=stub)

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
        # Clean up background tasks — always, even on assertion failure.
        stub.stop_signal = True  # exits action_loop
        event_queue.put(STOP_LOOP)  # exits event_send_loop
        stop_event.set()  # exits _wait_for_stop_event

        # Give tasks a moment to observe the signals, then cancel survivors.
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


# ---------------------------------------------------------------------------
# Test 4: event_queue drains before subprocess function exits
# ---------------------------------------------------------------------------


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
    stub_listener = _StubActionListener()

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
        type="STEP_EVENT_TYPE_COMPLETED",
        payload=None,
        should_not_retry=False,
    )
    event_queue.put(fake_event)

    # Allow the event_send_loop a moment to pick up the event.
    time.sleep(0.1)

    # Trigger graceful shutdown: stop action loop, then signal end of events.
    stop_event.set()  # stops action loop via _wait_for_stop_event
    stub_listener.stop_signal = True  # also stops action_loop iteration
    time.sleep(0.05)
    event_queue.put(STOP_LOOP)  # stops event_send_loop

    thread.join(timeout=15.0)
    assert not thread.is_alive(), "subprocess thread should have exited after STOP_LOOP"

    # The event_queue must be fully drained before the subprocess exited.
    assert (
        event_queue.empty()
    ), "event_queue must be fully drained before subprocess exits"
