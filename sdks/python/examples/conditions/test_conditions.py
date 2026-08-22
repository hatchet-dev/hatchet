import asyncio
import time

import pytest

from examples.conditions.worker import (
    cancel_if_event_workflow,
    cancel_if_or_workflow,
    cancel_if_sleep_workflow,
    cancel_if_workflow,
    skip_if_or_workflow,
    skip_if_sleep_workflow,
    task_condition_workflow,
    wait_for_event_only_workflow,
    sis_target,
    sio_target,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import Hatchet, RunStatus
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


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if(hatchet: Hatchet) -> None:
    ref = cancel_if_workflow.run(wait_for_result=False)

    await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)

    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_skip_if_sleep_skips_when_sleep_wins(hatchet: Hatchet) -> None:
    result = skip_if_sleep_workflow.run()

    assert result[sis_target.name] == {"skipped": True}


@pytest.mark.asyncio(loop_scope="session")
async def test_skip_if_sleep_runs_when_event_wins(hatchet: Hatchet) -> None:
    ref = skip_if_sleep_workflow.run(wait_for_result=False)

    await wait_for_running_status(hatchet, ref.workflow_run_id)

    # Push twice: the run can show RUNNING just before the engine registers
    # sis_target's wait_for match condition, and an unmatched event is dropped
    # silently. Both pushes land well inside the 12s skip_if sleep.
    for _ in range(2):
        await hatchet.event.aio_push("skip_if_sleep:proceed", {})
        await asyncio.sleep(2)

    result = await ref.aio_result()

    assert result[sis_target.name].get("skipped") is not True
    assert result[sis_target.name]["random_number"] == 2


@pytest.mark.asyncio(loop_scope="session")
async def test_skip_if_or_group_parent(hatchet: Hatchet) -> None:
    result = skip_if_or_workflow.run()

    assert result[sio_target.name] == {"skipped": True}


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_user_event(hatchet: Hatchet) -> None:
    ref = cancel_if_event_workflow.run(wait_for_result=False)

    await wait_for_running_status(hatchet, ref.workflow_run_id)

    # Push twice: the run can show RUNNING just before the engine registers
    # cie_target's cancel_if match condition, and an unmatched event is dropped
    # silently. Both pushes land well inside the 30s wait_for sleep.
    for _ in range(2):
        await hatchet.event.aio_push("cancel_if_event:abort", {})
        await asyncio.sleep(3)

    await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_sleep(hatchet: Hatchet) -> None:
    ref = cancel_if_sleep_workflow.run(wait_for_result=False)

    await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_or_group(hatchet: Hatchet) -> None:
    ref = cancel_if_or_workflow.run(wait_for_result=False)

    await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_wait_for_user_event(hatchet: Hatchet) -> None:
    ref = wait_for_event_only_workflow.run(wait_for_result=False)

    await wait_for_running_status(hatchet, ref.workflow_run_id)
    await asyncio.sleep(2)
    await hatchet.event.aio_push("wait_for_event_only:go", {})

    result = await ref.aio_result()

    assert result["wfe_target"]["random_number"] == 5
