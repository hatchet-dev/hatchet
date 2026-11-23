from uuid import uuid4

import pytest

from examples.run_details.worker import MockInput, run_detail_test_workflow
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet, test_run_id: str) -> None:
    mock_input = MockInput(foo=str(uuid4()))
    meta = {"test_run_id": test_run_id}
    ref = run_detail_test_workflow.run(
        input=mock_input,
        additional_metadata=meta,
        wait_for_result=False,
    )

    await wait_for_running_status(hatchet, ref.workflow_run_id, min_task_runs=3)
    details = await hatchet.runs.aio_get_details(ref.workflow_run_id)

    assert details.status == "RUNNING"
    assert details.input == mock_input.model_dump()
    assert details.additional_metadata is not None
    assert meta.items() <= details.additional_metadata.items()

    assert len(details.task_runs) == 4

    assert details.done is False

    with pytest.raises(FailedTaskRunExceptionGroup):
        await ref.aio_result()

    details = hatchet.runs.get_details(ref.workflow_run_id)

    assert details.status == "FAILED"
    assert details.input == mock_input.model_dump()
    assert details.additional_metadata is not None
    assert meta.items() <= details.additional_metadata.items()
    assert len(details.task_runs) == 6

    assert details.task_runs["step1"].status == "COMPLETED"
    assert details.task_runs["step2"].status == "COMPLETED"
    assert details.task_runs["step3"].status == "COMPLETED"
    assert details.task_runs["step4"].status == "COMPLETED"
    assert details.task_runs["fail_step"].status == "FAILED"
    assert details.task_runs["cancel_step"].status == "CANCELLED"

    step1_output = details.task_runs["step1"].output
    step2_output = details.task_runs["step2"].output
    step3_output = details.task_runs["step3"].output
    assert step1_output is not None
    assert step2_output is not None
    assert step3_output is not None
    assert (
        step1_output["random_number"] + step2_output["random_number"]
        == step3_output["sum"]
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
