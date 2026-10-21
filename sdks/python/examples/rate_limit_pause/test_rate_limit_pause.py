import asyncio
import time
from datetime import datetime, timedelta, timezone
from uuid import uuid4

import pytest

from examples.rate_limit_pause.worker import RateLimitPauseInput, rate_limit_pause_task
from hatchet_sdk import Hatchet, RunStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_rate_limit_pause(hatchet: Hatchet) -> None:
    group = f"rate-limit-pause-group-{uuid4()}"
    paused_key = f"rate-limit-pause-paused-{uuid4()}"
    unpaused_key = f"rate-limit-pause-unpaused-{uuid4()}"

    paused_until = await hatchet.rate_limits.aio_pause(
        paused_key, datetime.now(timezone.utc) + timedelta(seconds=15)
    )

    # both runs share a concurrency group with max_runs=1, so the paused run must not take the group's only slot
    paused_ref = await rate_limit_pause_task.aio_run(
        RateLimitPauseInput(group=group, key=paused_key),
        wait_for_result=False,
    )
    unpaused_ref = await rate_limit_pause_task.aio_run(
        RateLimitPauseInput(group=group, key=unpaused_key),
        wait_for_result=False,
    )

    await asyncio.wait_for(unpaused_ref.aio_result(), timeout=10)

    paused_details = await hatchet.runs.aio_get_details(paused_ref.workflow_run_id)
    assert (
        paused_details.status == RunStatus.QUEUED
    ), f"Run with a paused key is not queued (status {paused_details.status})."

    rate_limits = await hatchet.rate_limits.aio_list(search=paused_key)
    assert rate_limits.rows is not None and len(rate_limits.rows) == 1
    assert rate_limits.rows[0].paused_until is not None
    assert (
        rate_limits.rows[0].value == 10
    ), f"A queued run consumed units of a paused key (value {rate_limits.rows[0].value})."

    while datetime.now(timezone.utc) < paused_until - timedelta(seconds=1):
        paused_details = await hatchet.runs.aio_get_details(paused_ref.workflow_run_id)
        assert (
            paused_details.status == RunStatus.QUEUED
        ), f"Run with a paused key left the queue before the pause ended (status {paused_details.status})."

        await asyncio.sleep(1)

    # released items are requeued by a periodic poll, so this needs more than a few seconds
    timeout = 60
    start = time.time()

    while True:
        if time.time() - start > timeout:
            assert (
                False
            ), f"Run with a paused key did not complete within {timeout} seconds of the pause ending."

        paused_details = await hatchet.runs.aio_get_details(paused_ref.workflow_run_id)

        if paused_details.status == RunStatus.COMPLETED:
            break

        assert paused_details.status in [
            RunStatus.QUEUED,
            RunStatus.RUNNING,
        ], f"Run with a paused key is not queued or running (status {paused_details.status})."

        await asyncio.sleep(1)

    rate_limits = await hatchet.rate_limits.aio_list(search=paused_key)
    assert rate_limits.rows is not None and len(rate_limits.rows) == 1
    assert rate_limits.rows[0].paused_until is None
    assert (
        rate_limits.rows[0].value == 9
    ), f"Run did not consume units after the pause ended (value {rate_limits.rows[0].value})."
