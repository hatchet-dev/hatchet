import asyncio
import time
from uuid import uuid4

import pytest

from examples.concurrency_cancel_newest.worker import (
    WorkflowInput,
    concurrency_cancel_newest_workflow,
)
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
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

    timeout = 30
    start = time.time()
    successful_status = None
    cancelled_statuses: list[V1TaskStatus] = []

    while time.time() - start < timeout:
        successful_status = hatchet.runs.get(to_run.workflow_run_id).run.status
        cancelled_statuses = [
            r.status
            for r in hatchet.runs.list(
                additional_metadata={"test_run_id": test_run_id}
            ).rows
            if r.metadata.id != to_run.workflow_run_id
        ]

        if (
            successful_status == V1TaskStatus.COMPLETED
            and len(cancelled_statuses) == 10
            and all(s == V1TaskStatus.CANCELLED for s in cancelled_statuses)
        ):
            return

        await asyncio.sleep(1)

    assert False, (
        f"Expected the first run to be COMPLETED and the other 10 to be CANCELLED within {timeout} seconds, "
        f"but got first={successful_status} and others={cancelled_statuses}"
    )
