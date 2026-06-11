"""Tests for the DurableEventListener ordered-release gate.

Completions carrying a satisfied_order must be released to user code strictly
in that order, gated on continuation park, so durable task replays reproduce
the original wake order (see A->B / C->D non-determinism).
"""

from __future__ import annotations

import asyncio
from unittest.mock import MagicMock

import pytest

from hatchet_sdk.clients.listeners.durable_event_listener import (
    DurableEventListener,
    DurableTaskEventLogEntryResult,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableEventLogEntryRef,
    DurableTaskEventLogEntryCompletedResponse,
    DurableTaskResponse,
)
from hatchet_sdk.exceptions import NonDeterminismError


def make_listener() -> DurableEventListener:
    config = MagicMock()
    config.token = "test-token"
    return DurableEventListener(config, MagicMock())


def entry_completed(
    task_id: str,
    invocation: int,
    branch_id: int,
    node_id: int,
    payload: bytes = b"{}",
    satisfied_order: int | None = None,
) -> DurableTaskResponse:
    completed = DurableTaskEventLogEntryCompletedResponse(
        ref=DurableEventLogEntryRef(
            durable_task_external_id=task_id,
            invocation_count=invocation,
            branch_id=branch_id,
            node_id=node_id,
        ),
        payload=payload,
    )
    if satisfied_order is not None:
        completed.satisfied_order = satisfied_order

    return DurableTaskResponse(entry_completed=completed)


def register(
    listener: DurableEventListener,
    task_id: str,
    invocation: int,
    branch_id: int,
    node_id: int,
) -> "asyncio.Future[DurableTaskEventLogEntryResult]":
    """Registers a parked waiter the same way wait_for_callback does."""
    key = (task_id, invocation, branch_id, node_id)
    future: asyncio.Future[DurableTaskEventLogEntryResult] = asyncio.Future()
    listener._pending_callbacks[key] = future
    listener._notify_parked((task_id, invocation))
    return future


async def test_holds_out_of_order_completion() -> None:
    listener = make_listener()

    fut_a = register(listener, "task", 1, 1, 1)
    fut_c = register(listener, "task", 1, 1, 2)

    # A's completion was stamped second but arrives first: held.
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, b'{"r": "a"}', satisfied_order=2)
    )
    assert not fut_a.done()

    # C's completion (order 1) arrives: released, wakes C's continuation.
    await listener._handle_response(
        entry_completed("task", 1, 1, 2, b'{"r": "c"}', satisfied_order=1)
    )
    assert fut_c.done()
    assert fut_c.result().payload == {"r": "c"}

    # gate stays closed for order 2 until C's continuation parks.
    assert not fut_a.done()

    # C's continuation spawns D and parks on its result: order 2 released.
    fut_d = register(listener, "task", 1, 1, 3)
    assert fut_a.done()
    assert fut_a.result().payload == {"r": "a"}
    assert not fut_d.done()


async def test_buffered_release_keeps_pumping() -> None:
    listener = make_listener()

    # the only parked waiter awaits the entry satisfied at order 2 (sequential
    # code awaiting A first while C completed first).
    fut_2 = register(listener, "task", 1, 1, 1)

    await listener._handle_response(
        entry_completed("task", 1, 1, 2, b'{"r": "c"}', satisfied_order=1)
    )
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, b'{"r": "a"}', satisfied_order=2)
    )

    # order 1 had no waiter -> buffered; order 2 released to the waiter.
    assert fut_2.done()
    assert fut_2.result().payload == {"r": "a"}
    assert ("task", 1, 1, 2) in listener._buffered_completions


async def test_redelivery_bypasses_gate() -> None:
    listener = make_listener()

    fut_1 = register(listener, "task", 1, 1, 1)
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=1)
    )
    assert fut_1.done()

    # gate is closed (woken continuation hasn't parked), but a re-delivery of
    # order 1 is delivered immediately.
    fut_retry = register(listener, "task", 1, 1, 1)
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=1)
    )
    assert fut_retry.done()


async def test_legacy_completion_released_immediately() -> None:
    listener = make_listener()

    fut_ordered = register(listener, "task", 1, 1, 1)
    fut_legacy = register(listener, "task", 1, 1, 9)

    # ordered completion with a gap (order 2, order 1 missing): held.
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=2)
    )
    assert not fut_ordered.done()

    # legacy completion with no satisfied_order: delivered immediately.
    await listener._handle_response(entry_completed("task", 1, 1, 9))
    assert fut_legacy.done()


async def test_gap_timeout_fails_waiters() -> None:
    listener = make_listener()
    listener._gap_timeout_s = 0.0

    fut = register(listener, "task", 1, 1, 1)

    # order 2 arrives, order 1 never does (history diverged).
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=2)
    )
    assert not fut.done()

    listener._sweep_gates()

    assert fut.done()
    with pytest.raises(NonDeterminismError):
        fut.result()

    assert not listener._gates


async def test_park_timeout_forces_gate_open() -> None:
    listener = make_listener()
    listener._park_timeout_s = 0.0

    fut_1 = register(listener, "task", 1, 1, 1)
    fut_2 = register(listener, "task", 1, 1, 2)

    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=1)
    )
    assert fut_1.done()

    await listener._handle_response(
        entry_completed("task", 1, 1, 2, satisfied_order=2)
    )
    assert not fut_2.done()

    # the woken continuation never parks: park timeout forces the gate open.
    listener._sweep_gates()
    assert fut_2.done()


async def test_cleanup_drops_gates() -> None:
    listener = make_listener()

    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=2)
    )
    assert listener._gates

    listener.cleanup_task_state("task", 1)
    assert not listener._gates


async def test_gates_scoped_per_invocation() -> None:
    listener = make_listener()

    fut_inv1 = register(listener, "task", 1, 1, 1)
    fut_inv2 = register(listener, "task", 2, 1, 1)

    # invocation 1 is blocked on a gap.
    await listener._handle_response(
        entry_completed("task", 1, 1, 1, satisfied_order=2)
    )
    assert not fut_inv1.done()

    # invocation 2's order 1 releases independently.
    await listener._handle_response(
        entry_completed("task", 2, 1, 1, satisfied_order=1)
    )
    assert fut_inv2.done()
