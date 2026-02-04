import asyncio
import time
from uuid import uuid4

import pytest

from examples.concurrency_cancel_in_progress.worker import (
    WorkflowInput, concurrency_cancel_in_progress_workflow)
from hatchet_sdk import (Hatchet, TriggerWorkflowOptions, V1TaskStatus,
                         WorkflowRunRef)


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    refs: list[WorkflowRunRef] = []

    for i in range(10):
        ref = await concurrency_cancel_in_progress_workflow.aio_run_no_wait(
            WorkflowInput(group="A"),
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id, "i": str(i)},
            ),
        )
        refs.append(ref)
        await asyncio.sleep(1)

    for ref in refs:
        print(f"Waiting for run {ref.workflow_run_id} to complete")
        try:
            await ref.aio_result()
        except Exception:
            continue

    ## wait for the olap repo to catch up
    await asyncio.sleep(5)

    runs = sorted(
        hatchet.runs.list(additional_metadata={"test_run_id": test_run_id}).rows,
        key=lambda r: int((r.additional_metadata or {}).get("i", "0")),
    )

    assert len(runs) == 10
    assert (runs[-1].additional_metadata or {}).get("i") == "9"
    assert runs[-1].status == V1TaskStatus.COMPLETED
    assert all(r.status == V1TaskStatus.CANCELLED for r in runs[:-1])
