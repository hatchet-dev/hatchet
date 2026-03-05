from __future__ import annotations

import asyncio
import time
from typing import Any

import pytest

from examples.durable_complex.conftest import (
    assert_evicted,
    get_task_output,
    requires_durable_eviction,
)
from examples.durable_complex.rate_limit.worker import (
    DynamicRateLimitInput,
    RATE_LIMIT_KEY,
    durable_rate_limit_dynamic_workflow,
    durable_rate_limit_workflow,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.rate_limit import RateLimitDuration


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_rate_limit(hatchet: Hatchet) -> None:
    """Durable task with rate limit: run 3 workflows, all complete (rate limit throttles); evicted."""
    hatchet.rate_limits.put(RATE_LIMIT_KEY, 2, RateLimitDuration.SECOND)

    start = time.time()
    refs = await asyncio.gather(
        *[durable_rate_limit_workflow.aio_run_no_wait() for _ in range(3)]
    )

    await assert_evicted(hatchet, refs[0].workflow_run_id)

    results: list[dict[str, Any]] = await asyncio.gather(
        *[ref.aio_result() for ref in refs]
    )
    elapsed = time.time() - start

    assert len(results) == 3
    for r in results:
        out = get_task_output(
            r,
            "durable_rate_limit_task",
            "durableratelimitworkflow:durable_rate_limit_task",
        )
        assert out.get("status") == "completed"
    assert elapsed >= 2.5


@requires_durable_eviction
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_rate_limit_dynamic_key(hatchet: Hatchet) -> None:
    """Durable task with dynamic rate limit key from input; evicted during sleep."""
    dynamic_key = "dynamic-rate-limit-test"
    hatchet.rate_limits.put(dynamic_key, 2, RateLimitDuration.SECOND)

    ref = durable_rate_limit_dynamic_workflow.run_no_wait(
        DynamicRateLimitInput(group=dynamic_key)
    )
    await assert_evicted(hatchet, ref.workflow_run_id)
    result: dict[str, Any] = await ref.aio_result()
    out = get_task_output(
        result,
        "durable_rate_limit_dynamic_task",
        "durableratelimitdynamicworkflow:durable_rate_limit_dynamic_task",
    )
    assert out.get("status") == "completed"
