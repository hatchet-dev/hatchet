import asyncio
from uuid import uuid4

import pytest

from examples.bug_tests.payload_bug_on_replay.worker import (
    Input, StepOutput, payload_initial_cancel_bug_workflow, step1, step2)
from hatchet_sdk import (EmptyModel, Hatchet, TriggerWorkflowOptions,
                         V1TaskStatus)


@pytest.mark.asyncio(loop_scope="session")
async def test_payload_replay_bug(hatchet: Hatchet) -> None:
    """
    Tests the case where a task is initially inserted in a non-queued state (e.g. cancelled),
    but then is replayed. The task should initially have a null payload, but on replay the payload
    should be updated.
    """

    test_run_id = str(uuid4())

    ref = await payload_initial_cancel_bug_workflow.aio_run_no_wait(
        input=Input(random_number=42),
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id}
        ),
    )

    result = await ref.aio_result()

    step_1_output = StepOutput.model_validate(result[step1.name])

    assert step_1_output.should_cancel is True

    await asyncio.sleep(3)

    run = await hatchet.runs.aio_get(ref.workflow_run_id)

    tasks = sorted(run.tasks, key=lambda t: t.metadata.created_at)

    assert len(tasks) == 2

    assert tasks[0].status == V1TaskStatus.COMPLETED
    assert tasks[1].status == V1TaskStatus.CANCELLED

    await hatchet.runs.aio_replay(run_id=ref.workflow_run_id)
    await asyncio.sleep(3)

    result = await ref.aio_result()

    step_1_output = StepOutput.model_validate(result[step1.name])
    step_2_output = StepOutput.model_validate(result[step2.name])

    assert step_1_output.should_cancel is False
    assert step_2_output.should_cancel is False

    await asyncio.sleep(3)

    run = await hatchet.runs.aio_get(ref.workflow_run_id)

    tasks = sorted(run.tasks, key=lambda t: t.metadata.created_at)

    assert len(tasks) == 2
    assert tasks[0].status == V1TaskStatus.COMPLETED
    assert tasks[1].status == V1TaskStatus.COMPLETED
