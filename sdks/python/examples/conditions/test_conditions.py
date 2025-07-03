import asyncio

import pytest

from examples.conditions.worker import task_condition_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_waits(hatchet: Hatchet) -> None:
    ref = task_condition_workflow.run_no_wait()

    await asyncio.sleep(15)

    hatchet.event.push("skip_on_event:skip", {})
    hatchet.event.push("wait_for_event:start", {})

    result = await ref.aio_result()

    assert result["skip_on_event"] == {"skipped": True}

    first_random_number = result["start"]["random_number"]
    wait_for_event_random_number = result["wait_for_event"]["random_number"]
    wait_for_sleep_random_number = result["wait_for_sleep"]["random_number"]

    left_branch = result["left_branch"]
    right_branch = result["right_branch"]

    assert left_branch.get("skipped") is True or right_branch.get("skipped") is True

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
