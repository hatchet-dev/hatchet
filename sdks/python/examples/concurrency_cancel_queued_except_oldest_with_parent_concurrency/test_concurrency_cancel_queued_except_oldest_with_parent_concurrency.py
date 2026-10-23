import asyncio
from uuid import uuid4

import pytest

from examples.concurrency_cancel_queued_except_oldest_with_parent_concurrency.worker import (
    WorkflowInput,
    concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow,
)
from hatchet_sdk import Hatchet, RunStatus, WorkflowRunRef


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_queued_except_oldest_keeps_oldest_with_parent_concurrency(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    group = str(uuid4())

    workflow = concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow

    ## occupies the task-level concurrency slot while the rest of the runs pile up in the queue
    occupying_run = await workflow.aio_run(
        WorkflowInput(group=group),
        additional_metadata={"test_run_id": test_run_id},
        wait_for_result=False,
    )
    await asyncio.sleep(1)

    queued_refs: list[WorkflowRunRef] = []
    for _ in range(10):
        ref = await workflow.aio_run(
            WorkflowInput(group=group),
            additional_metadata={"test_run_id": test_run_id},
            wait_for_result=False,
        )
        queued_refs.append(ref)
        await asyncio.sleep(0.2)

    try:
        await occupying_run.aio_result()
    except Exception:
        pass

    for ref in queued_refs:
        try:
            await ref.aio_result()
        except Exception:
            pass

    occupying_details = await hatchet.runs.aio_get_details(
        occupying_run.workflow_run_id
    )
    assert occupying_details.status == RunStatus.COMPLETED

    oldest_queued, *superseded = queued_refs

    oldest_details = await hatchet.runs.aio_get_details(oldest_queued.workflow_run_id)
    assert oldest_details.status == RunStatus.COMPLETED

    superseded_details = [
        await hatchet.runs.aio_get_details(ref.workflow_run_id) for ref in superseded
    ]
    assert all(r.status == RunStatus.CANCELLED for r in superseded_details)
