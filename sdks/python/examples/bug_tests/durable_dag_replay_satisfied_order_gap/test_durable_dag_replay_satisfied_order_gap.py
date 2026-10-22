import asyncio

import pytest

from examples.bug_tests.durable_dag_replay_satisfied_order_gap.worker import replay_gap
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet, RunStatus, V1TaskStatus
from hatchet_sdk.clients.admin import WorkflowRunDetail


async def wait_for_done(
    hatchet: Hatchet, run_id: str, timeout: float
) -> WorkflowRunDetail:
    deadline = asyncio.get_running_loop().time() + timeout

    while True:
        details = await hatchet.runs.aio_get_details(run_id)

        if details.done:
            return details

        if asyncio.get_running_loop().time() >= deadline:
            break

        await asyncio.sleep(0.5)

    raise TimeoutError(
        f"run {run_id} is still {details.status} after {timeout}s: "
        f"{ {name: task.status for name, task in details.task_runs.items()} }"
    )


@pytest.mark.timeout(240, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_completes_after_replaying_a_failed_step_whose_sibling_finished_later(
    hatchet: Hatchet,
) -> None:
    ref = await replay_gap.aio_run(wait_for_result=False)

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    details = await wait_for_done(hatchet, ref.workflow_run_id, timeout=60)

    assert details.task_runs["flaky"].status == V1TaskStatus.FAILED
    assert details.task_runs["slow"].status == V1TaskStatus.COMPLETED

    await hatchet.runs.aio_replay(details.task_runs["flaky"].external_id)
    await wait_for_running_status(hatchet, ref.workflow_run_id)

    details = await wait_for_done(hatchet, ref.workflow_run_id, timeout=150)

    assert details.status == RunStatus.COMPLETED
    assert details.task_runs["flaky"].output == {"retry_count": 1}
    assert details.task_runs["quick"].output == {"retry_count": 0}
    assert details.task_runs["slow"].output == {"retry_count": 0}
