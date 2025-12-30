import asyncio
from uuid import uuid4

import pytest

from examples.run_details.worker import MockInput, hatchet, run_detail_test_workflow
from hatchet_sdk import Hatchet, V1TaskStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    mock_input = MockInput(foo=str(uuid4()))
    ref = run_detail_test_workflow.run_no_wait(input=mock_input)

    await asyncio.sleep(2)

    payloads = hatchet.runs.get_details(ref.workflow_run_id)

    assert payloads.status == V1TaskStatus.RUNNING
    assert payloads.input == mock_input.model_dump()
    assert len(payloads.task_runs) == 4
    assert all(
        r.status in [V1TaskStatus.RUNNING, V1TaskStatus.QUEUED]
        for r in payloads.task_runs.values()
    )
    assert all(r.error is None for r in payloads.task_runs.values())
    assert all(r.output is None for r in payloads.task_runs.values())
    assert "step3" not in payloads.task_runs
    assert "step4" not in payloads.task_runs

    try:
        await ref.aio_result()
    except Exception:
        pass

    payloads = hatchet.runs.get_details(ref.workflow_run_id)

    assert payloads.status == V1TaskStatus.FAILED
    assert payloads.input == mock_input.model_dump()
    assert len(payloads.task_runs) == 6

    assert payloads.task_runs["step1"].status == V1TaskStatus.COMPLETED
    assert payloads.task_runs["step2"].status == V1TaskStatus.COMPLETED
    assert payloads.task_runs["step3"].status == V1TaskStatus.COMPLETED
    assert payloads.task_runs["step4"].status == V1TaskStatus.COMPLETED
    assert payloads.task_runs["fail_step"].status == V1TaskStatus.FAILED
    assert payloads.task_runs["cancel_step"].status == V1TaskStatus.CANCELLED

    assert (
        payloads.task_runs["step1"].output["random_number"]  # type: ignore[index]
        + payloads.task_runs["step2"].output["random_number"]  # type: ignore[index]
        == payloads.task_runs["step3"].output["sum"]  # type: ignore[index]
    )

    assert payloads.task_runs["step4"].output == {"step4": "step4"}

    assert payloads.task_runs["fail_step"].error is not None
    assert payloads.task_runs["fail_step"].output is None

    assert payloads.task_runs["cancel_step"].error is None
    assert payloads.task_runs["cancel_step"].output is None

    assert all(
        r.error is None
        for name, r in payloads.task_runs.items()
        if name not in ["fail_step"]
    )
