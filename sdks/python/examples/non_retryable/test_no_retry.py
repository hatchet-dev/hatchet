import pytest

from examples.non_retryable.worker import (
    non_retryable_workflow,
    should_not_retry,
    should_not_retry_successful_task,
    should_retry_wrong_exception_type,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails


def find_id(runs: V1WorkflowRunDetails, match: str) -> str:
    return next(t.metadata.id for t in runs.tasks if match in t.display_name)


@pytest.mark.asyncio()
async def test_no_retry(hatchet: Hatchet) -> None:
    ref = await non_retryable_workflow.aio_run_no_wait()

    with pytest.raises(Exception, match="retry"):
        await ref.aio_result()

    runs = await hatchet.runs.aio_get(ref.workflow_run_id)
    task_to_id = {
        task: find_id(runs, task.name)
        for task in [
            should_not_retry_successful_task,
            should_retry_wrong_exception_type,
            should_not_retry,
        ]
    }

    retrying_events = [
        e for e in runs.task_events if e.event_type == V1TaskEventType.RETRYING
    ]

    """Only one task should be retried."""
    assert len(retrying_events) == 1

    """The task id of the retrying events should match the tasks that are retried"""
    assert {e.task_id for e in retrying_events} == {
        task_to_id[should_retry_wrong_exception_type],
    }

    """Three failed events should emit, one each for the two failing initial runs and one for the retry."""
    assert (
        len([e for e in runs.task_events if e.event_type == V1TaskEventType.FAILED])
        == 3
    )
