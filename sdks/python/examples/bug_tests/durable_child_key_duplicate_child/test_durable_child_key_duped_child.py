import pytest

from examples.bug_tests.durable_child_key_duplicate_child.worker import (
    durable_parent_child_key_bug,
)
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_child_key_duplicate_bug(hatchet: Hatchet) -> None:
    res = await durable_parent_child_key_bug.aio_run(wait_for_result=False)
    run_id = res.workflow_run_id

    await res.aio_result()

    runs = await hatchet.runs.aio_list(parent_task_external_id=run_id)

    assert (
        len(runs.rows) == 1
    ), "should only have one child since the `child_key` is set"

    run = runs.rows[0]

    assert run.status == V1TaskStatus.COMPLETED
