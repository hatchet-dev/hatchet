import asyncio

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


@pytest.mark.asyncio(loop_scope="session")
async def test_waits(hatchet: Hatchet) -> None:
    ref = task_condition_workflow.run(wait_for_result=False)

    await wait_for_running_status(hatchet, ref.workflow_run_id)
    await asyncio.sleep(15)

    hatchet.event.push("skip_on_event:skip", {})
    hatchet.event.push("wait_for_event:start", {})
    await asyncio.sleep(5)

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

    with pytest.raises(Exception):
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
    await asyncio.sleep(3)

    await hatchet.event.aio_push("skip_if_sleep:proceed", {})

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
    await asyncio.sleep(3)
    hatchet.event.push("cancel_if_event:abort", {})

    with pytest.raises(Exception):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_sleep(hatchet: Hatchet) -> None:
    ref = cancel_if_sleep_workflow.run(wait_for_result=False)

    with pytest.raises(Exception):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.status == RunStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_or_group(hatchet: Hatchet) -> None:
    ref = cancel_if_or_workflow.run(wait_for_result=False)

    with pytest.raises(Exception):
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
