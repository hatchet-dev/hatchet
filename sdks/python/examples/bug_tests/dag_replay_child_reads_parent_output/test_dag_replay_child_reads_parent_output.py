import asyncio

import pytest

from examples.bug_tests.dag_replay_child_reads_parent_output.worker import (
    child_reading_parent_output,
    dag_replay_child_reads_parent_output,
    parent_failing_on_first_attempt,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet, RunStatus, V1TaskStatus


async def wait_for_run_status(
    hatchet: Hatchet, run_id: str, status: RunStatus, timeout: float = 60.0
) -> None:
    interval = 0.5

    for _ in range(int(timeout / interval)):
        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == status:
            return

        await asyncio.sleep(interval)

    raise TimeoutError(f"run {run_id} did not reach {status}")


async def run_until_parent_fails(hatchet: Hatchet) -> str:
    ref = await dag_replay_child_reads_parent_output.aio_run(wait_for_result=False)

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)

    assert (
        details.task_runs[parent_failing_on_first_attempt.name].status
        == V1TaskStatus.FAILED
    )
    assert (
        details.task_runs[child_reading_parent_output.name].status
        == V1TaskStatus.CANCELLED
    )

    return ref.workflow_run_id


async def replay_and_assert_child_received_parent_output(
    hatchet: Hatchet,
    run_id: str,
    replay_target_id: str,
    expected_parent_retry_count: int,
    expected_child_retry_count: int,
) -> None:
    await hatchet.runs.aio_replay(replay_target_id)
    await wait_for_running_status(hatchet, run_id)
    await wait_for_run_status(hatchet, run_id, RunStatus.COMPLETED, timeout=90)

    details = await hatchet.runs.aio_get_details(run_id)

    assert details.task_runs[child_reading_parent_output.name].output == {
        "parent_retry_count": expected_parent_retry_count,
        "retry_count": expected_child_retry_count,
    }


@pytest.mark.timeout(120, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_child_reads_parent_output_after_replaying_the_failed_parent(
    hatchet: Hatchet,
) -> None:
    run_id = await run_until_parent_fails(hatchet)
    details = await hatchet.runs.aio_get_details(run_id)

    await replay_and_assert_child_received_parent_output(
        hatchet,
        run_id,
        details.task_runs[parent_failing_on_first_attempt.name].external_id,
        expected_parent_retry_count=1,
        expected_child_retry_count=1,
    )


@pytest.mark.timeout(120, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_child_reads_parent_output_after_replaying_the_whole_run(
    hatchet: Hatchet,
) -> None:
    run_id = await run_until_parent_fails(hatchet)

    await replay_and_assert_child_received_parent_output(
        hatchet,
        run_id,
        run_id,
        expected_parent_retry_count=1,
        expected_child_retry_count=1,
    )
