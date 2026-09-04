import pytest

from examples.bug_tests.durable_child_key_duplicate_child.worker import (
    durable_parent_child_key_bug,
    Input,
)
from examples.test_utils import poll_for_runs
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_key_duplicate_bug_all_duped(hatchet: Hatchet) -> None:
    res = await durable_parent_child_key_bug.aio_run(
        input=Input(scenario="all_duped"), wait_for_result=False
    )

    await res.aio_result()

    runs = await poll_for_runs(
        hatchet, expected_count=1, parent_task_external_id=res.workflow_run_id
    )

    assert len(runs) == 1, "should only have one child since the `child_key` is set"

    run = runs[0]

    assert run.status == V1TaskStatus.COMPLETED


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_key_duplicate_bug_second_unique(hatchet: Hatchet) -> None:
    res = await durable_parent_child_key_bug.aio_run(
        input=Input(scenario="second_unique"), wait_for_result=False
    )

    await res.aio_result()

    runs = await poll_for_runs(
        hatchet, expected_count=2, parent_task_external_id=res.workflow_run_id
    )

    assert (
        len(runs) == 2
    ), "should have two children since the second `child_key` is unique"

    first, second = runs

    assert first.status == V1TaskStatus.COMPLETED
    assert second.status == V1TaskStatus.COMPLETED

    assert (
        first.workflow_run_external_id != second.workflow_run_external_id
    ), "second should be different than first"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_key_duplicate_bug_third_unique(hatchet: Hatchet) -> None:
    res = await durable_parent_child_key_bug.aio_run(
        input=Input(scenario="third_unique"), wait_for_result=False
    )

    await res.aio_result()

    runs = await poll_for_runs(
        hatchet, expected_count=2, parent_task_external_id=res.workflow_run_id
    )

    assert (
        len(runs) == 2
    ), "should only have two children since only the third `child_key` is unique"

    first, second = runs

    assert first.status == V1TaskStatus.COMPLETED
    assert second.status == V1TaskStatus.COMPLETED

    assert (
        first.workflow_run_external_id != second.workflow_run_external_id
    ), "second should be different than first"
