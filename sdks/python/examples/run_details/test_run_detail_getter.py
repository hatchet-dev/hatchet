import asyncio
from uuid import uuid4

import pytest

from examples.run_details.worker import MockInput, run_detail_test_workflow
from hatchet_sdk import (Hatchet, RunStatus, TriggerWorkflowOptions,
                         V1TaskStatus)


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    mock_input = MockInput(foo=str(uuid4()))
    test_run_id = str(uuid4())
    meta = {"test_run_id": test_run_id}
    ref = run_detail_test_workflow.run_no_wait(
        input=mock_input,
        options=TriggerWorkflowOptions(additional_metadata=meta),
    )

    await asyncio.sleep(2)

    details = hatchet.runs.get_details(ref.workflow_run_id)

    assert details.status == RunStatus.RUNNING
    assert details.input == mock_input.model_dump()
    assert details.additional_metadata == meta
    assert len(details.task_runs) == 4
    assert all(
        r.status in [V1TaskStatus.RUNNING, V1TaskStatus.QUEUED]
        for r in details.task_runs.values()
    )
    assert all(r.error is None for r in details.task_runs.values())
    assert all(r.output is None for r in details.task_runs.values())
    assert "step3" not in details.task_runs
    assert "step4" not in details.task_runs
    assert details.done is False

    with pytest.raises(Exception):
        await ref.aio_result()

    details = hatchet.runs.get_details(ref.workflow_run_id)

    assert details.status == RunStatus.FAILED
    assert details.input == mock_input.model_dump()
    assert details.additional_metadata == meta
    assert len(details.task_runs) == 6

    assert details.task_runs["step1"].status == V1TaskStatus.COMPLETED
    assert details.task_runs["step2"].status == V1TaskStatus.COMPLETED
    assert details.task_runs["step3"].status == V1TaskStatus.COMPLETED
    assert details.task_runs["step4"].status == V1TaskStatus.COMPLETED
    assert details.task_runs["fail_step"].status == V1TaskStatus.FAILED
    assert details.task_runs["cancel_step"].status == V1TaskStatus.CANCELLED

    assert (
        details.task_runs["step1"].output["random_number"]  # type: ignore[index]
        + details.task_runs["step2"].output["random_number"]  # type: ignore[index]
        == details.task_runs["step3"].output["sum"]  # type: ignore[index]
    )

    assert details.task_runs["step4"].output == {"step4": "step4"}

    assert details.task_runs["fail_step"].error is not None
    assert details.task_runs["fail_step"].output is None

    assert details.task_runs["cancel_step"].error is None
    assert details.task_runs["cancel_step"].output is None

    assert all(
        r.error is None
        for name, r in details.task_runs.items()
        if name not in ["fail_step"]
    )

    assert details.done is True
