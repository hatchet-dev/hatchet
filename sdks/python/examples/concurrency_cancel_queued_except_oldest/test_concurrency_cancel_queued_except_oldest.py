import asyncio
from uuid import uuid4

import pytest

from examples.concurrency_cancel_queued_except_oldest.worker import (
    WorkflowInput,
    concurrency_cancel_queued_except_oldest_workflow,
)
from hatchet_sdk import Hatchet, V1TaskStatus, WorkflowRunRef


@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_queued_except_oldest_keeps_only_the_oldest_queued_run(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    group = str(uuid4())

    ## occupies the concurrency slot for the group while the rest of the runs pile up in the queue
    occupying_run = await concurrency_cancel_queued_except_oldest_workflow.aio_run(
        WorkflowInput(group=group),
        additional_metadata={"test_run_id": test_run_id, "i": "0"},
        wait_for_result=False,
    )

    await asyncio.sleep(1)

    queued_refs: list[WorkflowRunRef] = []
    for i in range(1, 11):
        ref = await concurrency_cancel_queued_except_oldest_workflow.aio_run(
            WorkflowInput(group=group),
            additional_metadata={"test_run_id": test_run_id, "i": str(i)},
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

    ## wait for the olap repo to catch up
    await asyncio.sleep(5)

    runs = sorted(
        hatchet.runs.list(additional_metadata={"test_run_id": test_run_id}),
        key=lambda r: int((r.additional_metadata or {}).get("i", "0")),
    )

    assert len(runs) == 11

    occupying, oldest_queued, *superseded = runs

    assert occupying.status == V1TaskStatus.COMPLETED

    assert (oldest_queued.additional_metadata or {}).get("i") == "1"
    assert oldest_queued.status == V1TaskStatus.COMPLETED

    assert all(r.status == V1TaskStatus.CANCELLED for r in superseded)
