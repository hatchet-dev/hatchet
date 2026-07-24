from __future__ import annotations

from datetime import timedelta
from typing import Any, cast
from unittest.mock import AsyncMock

import pytest

from hatchet_sdk.context.context import DurableContext, _first_wait_match_or_none


@pytest.mark.parametrize(
    "result",
    [
        {},
        {"CREATE": {}},
        {"CREATE": {"signal_key_1": []}},
    ],
)
def test_first_wait_match_or_none_returns_none_for_empty_payloads(
    result: dict[str, Any],
) -> None:
    assert _first_wait_match_or_none(result) is None


def test_first_wait_match_or_none_extracts_first_match() -> None:
    result = {"CREATE": {"signal_key_1": [{"id": "abc", "sleep_duration": "5s"}]}}

    assert _first_wait_match_or_none(result) == {"id": "abc", "sleep_duration": "5s"}


def _context_with_wait_for(aio_wait_for_result: dict[str, Any]) -> DurableContext:
    ctx = DurableContext.__new__(DurableContext)
    ctx._wait_index = 0
    ctx.aio_wait_for = AsyncMock(return_value=aio_wait_for_result)  # type: ignore[method-assign]

    return ctx


async def test_aio_sleep_for_does_not_crash_on_empty_payload() -> None:
    """
    Regression test for issue #4349: a durable wait that completes with an
    empty CREATE payload used to raise `RuntimeError: coroutine raised
    StopIteration` (PEP 479) instead of falling back gracefully.
    """
    ctx = _context_with_wait_for({"CREATE": {}})

    result = await ctx.aio_sleep_for(timedelta(seconds=5))

    assert result.duration == timedelta(seconds=5)


async def test_aio_wait_for_event_does_not_crash_on_empty_payload() -> None:
    ctx = _context_with_wait_for({"CREATE": {}})
    ctx.aio_now = AsyncMock()  # type: ignore[method-assign]

    result = await ctx.aio_wait_for_event("some-event")

    assert result == {}


async def test_aio_wait_for_event_extracts_payload_when_present() -> None:
    ctx = _context_with_wait_for({"CREATE": {"event:some-event-0": [{"foo": "bar"}]}})
    ctx.aio_now = AsyncMock()  # type: ignore[method-assign]

    result = await ctx.aio_wait_for_event("some-event")

    assert result == cast(dict[str, Any], {"foo": "bar"})
