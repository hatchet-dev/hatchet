from __future__ import annotations

import asyncio
from typing import Any

import pytest

from examples.durable_event.worker import (
    EVENT_KEY,
    durable_event_task,
    durable_event_task_with_filter,
    durable_event_task_filter_mismatch,
)
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_wait(hatchet: Hatchet) -> None:
    """Task completes after matching event is pushed; evicted while waiting for event."""
    ref = durable_event_task.run_no_wait()

    async def push_event() -> None:
        await asyncio.sleep(5)
        hatchet.event.push(EVENT_KEY, {"user_id": "1234", "data": "test"})

    asyncio.create_task(push_event())
    result: dict[str, Any] = await ref.aio_result()
    assert isinstance(result, dict)
    assert "CREATE" in result
    event_payloads = result["CREATE"][EVENT_KEY]
    assert len(event_payloads) == 1
    assert event_payloads[0]["user_id"] == "1234"
    assert event_payloads[0]["data"] == "test"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_filter_match(hatchet: Hatchet) -> None:
    """Filtered task completes when filter expression matches; evicted while waiting."""
    ref = durable_event_task_with_filter.run_no_wait()

    async def push_event() -> None:
        await asyncio.sleep(5)
        hatchet.event.push(EVENT_KEY, {"user_id": "1234"})

    asyncio.create_task(push_event())
    result: dict[str, Any] = await ref.aio_result()
    assert "CREATE" in result
    assert EVENT_KEY in result["CREATE"]
    assert result["CREATE"][EVENT_KEY][0]["user_id"] == "1234"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_event_filter_mismatch(hatchet: Hatchet) -> None:
    """Filtered task keeps waiting when event does not match, then completes; evicted while waiting."""
    ref = durable_event_task_filter_mismatch.run_no_wait()

    async def push_matching_event() -> None:
        await asyncio.sleep(2)
        hatchet.event.push(EVENT_KEY, {"user_id": "1234"})
        await asyncio.sleep(4)
        hatchet.event.push(EVENT_KEY, {"user_id": "9999"})

    asyncio.create_task(push_matching_event())
    result: dict[str, Any] = await ref.aio_result()
    assert "CREATE" in result
    assert result["CREATE"][EVENT_KEY][0]["user_id"] == "9999"
