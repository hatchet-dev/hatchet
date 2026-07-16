import asyncio
from uuid import uuid4

import pytest

from examples.concurrency_cancel_newest_task_level.worker import (
    WorkflowInput,
    concurrency_cancel_newest_task_level_workflow,
)
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    """Task-level (step-level) CANCEL_NEWEST must keep the OLDEST run and cancel the newer ones,
    never preempting the running run. This exercises the in-memory concurrency index (the only path
    that serves step-level concurrency).

    Together with the CANCEL_IN_PROGRESS task-level test this pins the two strategies as opposites:
    CANCEL_IN_PROGRESS keeps the newest, CANCEL_NEWEST keeps the oldest.
    """
    test_run_id = str(uuid4())

    # this run starts first and, as the oldest, must be the one that completes.
    to_run = await concurrency_cancel_newest_task_level_workflow.aio_run(
        WorkflowInput(group="A"),
        additional_metadata={"test_run_id": test_run_id},
        wait_for_result=False,
    )
    await asyncio.sleep(1)

    # these arrive while the first run is in progress; CANCEL_NEWEST rejects them.
    to_cancel = await concurrency_cancel_newest_task_level_workflow.aio_run_many(
        [
            concurrency_cancel_newest_task_level_workflow.create_bulk_run_item(
                input=WorkflowInput(group="A"),
                additional_metadata={"test_run_id": test_run_id},
            )
            for _ in range(10)
        ],
        wait_for_result=False,
    )

    await to_run.aio_result()

    for ref in to_cancel:
        try:
            await ref.aio_result()
        except Exception:
            pass

    ## wait for the olap repo to catch up
    await asyncio.sleep(5)

    successful_run = hatchet.runs.get(to_run.workflow_run_id)

    assert successful_run.run.status == V1TaskStatus.COMPLETED
    assert all(
        r.status == V1TaskStatus.CANCELLED
        for r in hatchet.runs.list(
            additional_metadata={"test_run_id": test_run_id}
        ).rows
        if r.metadata.id != to_run.workflow_run_id
    )
