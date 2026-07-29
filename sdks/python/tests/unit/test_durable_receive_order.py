"""The engine delivers durable completions already in satisfied_order, so the
listener just resolves each completion's waiter (or buffers it for a waiter that
has not registered yet). These tests cover that receive-order behavior."""

from __future__ import annotations

import asyncio
from unittest.mock import MagicMock

from hatchet_sdk.clients.listeners.durable_event_listener import (
    DurableEventListener,
    DurableTaskEventLogEntryResult,
)
from hatchet_sdk.contracts.v1.dispatcher_pb2 import (
    DurableEventLogEntryRef,
    DurableTaskEventLogEntryCompletedResponse,
    DurableTaskResponse,
)


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
) -> DurableTaskResponse:
    return DurableTaskResponse(
        entry_completed=DurableTaskEventLogEntryCompletedResponse(
            ref=DurableEventLogEntryRef(
                durable_task_external_id=task_id,
                invocation_count=invocation,
                branch_id=branch_id,
                node_id=node_id,
            ),
            payload=payload,
        )
    )


def register(
    listener: DurableEventListener,
    task_id: str,
    invocation: int,
    branch: int,
    node: int,
) -> asyncio.Future[DurableTaskEventLogEntryResult]:
    key = (task_id, invocation, branch, node)
    future: asyncio.Future[DurableTaskEventLogEntryResult] = asyncio.Future()
    listener._pending_callbacks[key] = future
    return future


async def test_completion_resolves_registered_waiter() -> None:
    listener = make_listener()
    fut = register(listener, "task", 1, 1, 1)

    await listener._handle_response(entry_completed("task", 1, 1, 1, b'{"r": "a"}'))

    assert fut.done()
    assert fut.result().payload == {"r": "a"}


async def test_completion_before_registration_is_buffered() -> None:
    listener = make_listener()

    await listener._handle_response(entry_completed("task", 1, 1, 1, b'{"r": "a"}'))

    assert ("task", 1, 1, 1) in listener._buffered_completions
    assert not listener._pending_callbacks


async def test_each_completion_resolves_its_own_waiter_by_key() -> None:
    listener = make_listener()
    fut_1 = register(listener, "task", 1, 1, 1)
    fut_2 = register(listener, "task", 1, 1, 2)

    # completions delivered in receive order, each keyed by node id
    await listener._handle_response(entry_completed("task", 1, 1, 2, b'{"r": "c"}'))
    await listener._handle_response(entry_completed("task", 1, 1, 1, b'{"r": "a"}'))

    assert fut_1.result().payload == {"r": "a"}
    assert fut_2.result().payload == {"r": "c"}
