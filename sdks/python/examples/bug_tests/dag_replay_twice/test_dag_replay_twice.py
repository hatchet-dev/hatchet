import pytest

from examples.bug_tests.dag_replay_twice.worker import dag_replay_twice, first
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet, V1TaskStatus


@pytest.mark.timeout(60, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_dag_replayed_after_a_single_task_replay(hatchet: Hatchet) -> None:
    """
    The first step fails and the second is cancelled. The failed task alone is replayed and
    the DAG completes. Replaying the whole run afterwards fails inside the DAG operator with

        failed to trigger step ...: duplicate key value violates unique constraint
        "v1_lookup_table_..._external_id_uq"
    """

    ref = await dag_replay_twice.aio_run(wait_for_result=False)

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)
    assert details.task_runs["first"].status == V1TaskStatus.FAILED
    assert details.task_runs["second"].status == V1TaskStatus.CANCELLED

    await hatchet.runs.aio_replay(details.task_runs["first"].external_id)
    await wait_for_running_status(hatchet, ref.workflow_run_id)

    result = await ref.aio_result()
    assert result[first.name] == {"retry_count": 1}

    await hatchet.runs.aio_replay(ref.workflow_run_id)
    await wait_for_running_status(hatchet, ref.workflow_run_id)

    result = await ref.aio_result()
    assert result[first.name] == {"retry_count": 2}
