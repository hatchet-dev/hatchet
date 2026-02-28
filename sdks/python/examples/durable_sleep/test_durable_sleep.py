from __future__ import annotations

import time
from typing import Any

import pytest

from examples.durable_sleep.worker import durable_sleep_task, SLEEP_DURATION


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_sleep_completes() -> None:
    """Task completes after sleep duration; evicted during sleep."""
    start = time.time()
    result: dict[str, Any] = await durable_sleep_task.aio_run()
    elapsed = time.time() - start

    assert result is not None
    assert elapsed >= SLEEP_DURATION
