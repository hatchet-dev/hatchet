import asyncio

import pytest

from examples.concurrency_cancel_newest.worker import (
    WorkflowInput,
    concurrency_cancel_newest_workflow,
)
from examples.test_utils import poll_for_runs
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet, test_run_id: str) -> None:
    to_run = await concurrency_cancel_newest_workflow.aio_run(
        WorkflowInput(group="A"),
        additional_metadata={
            "test_run_id": test_run_id,
        },
        wait_for_result=False,
    )
    await asyncio.sleep(1)

    to_cancel = await concurrency_cancel_newest_workflow.aio_run_many(
        [
            concurrency_cancel_newest_workflow.create_bulk_run_item(
                input=WorkflowInput(group="A"),
                additional_metadata={
                    "test_run_id": test_run_id,
                },
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

    await poll_for_runs(
        hatchet,
        expected_count=11,
        additional_metadata={"test_run_id": test_run_id},
        statuses=[V1TaskStatus.COMPLETED, V1TaskStatus.CANCELLED],
    )

    successful_run = hatchet.runs.get(to_run.workflow_run_id)

    assert successful_run.run.status == V1TaskStatus.COMPLETED
    assert all(
        r.status == V1TaskStatus.CANCELLED
        for r in hatchet.runs.list(additional_metadata={"test_run_id": test_run_id})
        if r.metadata.id != to_run.workflow_run_id
    )
