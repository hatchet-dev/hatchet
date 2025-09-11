import asyncio
from uuid import uuid4

import pytest

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet, TriggerWorkflowOptions


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    ref = await parent_wf.aio_run_no_wait(
        ParentInput(n=2),
    )

    result = await ref.aio_result()

    assert len(result["spawn"]["results"]) == 2


@pytest.mark.asyncio(loop_scope="session")
async def test_additional_metadata_propagation(hatchet: Hatchet) -> None:
    test_run_id = uuid4().hex

    ref = await parent_wf.aio_run_no_wait(
        ParentInput(n=2),
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id}
        ),
    )

    await ref.aio_result()
    await asyncio.sleep(1)

    runs = await hatchet.runs.aio_list(
        parent_task_external_id=ref.workflow_run_id,
        additional_metadata={"test_run_id": test_run_id},
    )

    assert runs.rows

    """Assert that the additional metadata is propagated to the child runs."""
    for run in runs.rows:
        assert run.additional_metadata
        assert run.additional_metadata["test_run_id"] == test_run_id

        assert run.children
        for child in run.children:
            assert child.additional_metadata
            assert child.additional_metadata["test_run_id"] == test_run_id
