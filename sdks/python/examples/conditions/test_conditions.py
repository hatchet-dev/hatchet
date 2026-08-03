import asyncio
import time

import pytest

from examples.conditions.worker import task_condition_workflow
from hatchet_sdk import Hatchet, V1TaskStatus


async def _wait_for_start_to_complete(
    hatchet: Hatchet, workflow_run_id: str, timeout: float = 60.0
) -> None:
    interval = 0.5
    deadline = time.monotonic() + timeout

    while time.monotonic() < deadline:
        details = await hatchet.runs.aio_get(workflow_run_id)

        # a task's display name is "<step readable id>-<unix timestamp>"
        if any(
            t.status == V1TaskStatus.COMPLETED and t.display_name.startswith("start-")
            for t in details.tasks
        ):
            return

        await asyncio.sleep(interval)

    raise TimeoutError(
        f"start did not complete within {timeout}s for run {workflow_run_id}"
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_waits(hatchet: Hatchet) -> None:
    ref = task_condition_workflow.run(wait_for_result=False)

    # skip_on_event's skip match and its 30s sleep are both registered when start
    # completes, so the skip event only counts during the 30s after that.
    await _wait_for_start_to_complete(hatchet, ref.workflow_run_id)

    # Push twice: the read model can show start COMPLETED just before the engine
    # registers the match, and an unmatched event is dropped silently.
    for _ in range(2):
        await hatchet.event.aio_push("skip_on_event:skip", {})
        await hatchet.event.aio_push("wait_for_event:start", {})
        await asyncio.sleep(2)

    result = await ref.aio_result()

    assert result["skip_on_event"] == {"skipped": True}

    first_random_number = result["start"]["random_number"]
    wait_for_event_random_number = result["wait_for_event"]["random_number"]
    wait_for_sleep_random_number = result["wait_for_sleep"]["random_number"]

    left_branch = result["left_branch"]
    right_branch = result["right_branch"]

    assert left_branch.get("skipped") is True or right_branch.get("skipped") is True

    skip_with_multiple_parents = result["skip_with_multiple_parents"]

    assert skip_with_multiple_parents.get("skipped") is True

    branch_random_number = left_branch.get("random_number") or right_branch.get(
        "random_number"
    )

    result_sum = result["sum"]["sum"]

    assert (
        result_sum
        == first_random_number
        + wait_for_event_random_number
        + wait_for_sleep_random_number
        + branch_random_number
    )
