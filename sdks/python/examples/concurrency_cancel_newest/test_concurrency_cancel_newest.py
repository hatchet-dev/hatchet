import asyncio
import time
from uuid import uuid4

import pytest

from examples.concurrency_cancel_newest.worker import (
    WorkflowInput,
    concurrency_cancel_newest_workflow,
)
from hatchet_sdk import Hatchet, TriggerWorkflowOptions, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    to_run = await concurrency_cancel_newest_workflow.aio_run_no_wait(
        WorkflowInput(group="A"),
        options=TriggerWorkflowOptions(
            additional_metadata={
                "test_run_id": test_run_id,
            },
        ),
    )
    await asyncio.sleep(1)

    to_cancel = await concurrency_cancel_newest_workflow.aio_run_many_no_wait(
        [
            concurrency_cancel_newest_workflow.create_bulk_run_item(
                input=WorkflowInput(group="A"),
                options=TriggerWorkflowOptions(
                    additional_metadata={
                        "test_run_id": test_run_id,
                    },
                ),
            )
            for _ in range(10)
        ]
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
