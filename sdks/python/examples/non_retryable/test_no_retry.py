import asyncio

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


@pytest.mark.asyncio(loop_scope="session")
async def test_no_retry(hatchet: Hatchet) -> None:
    ref = await non_retryable_workflow.aio_run_no_wait()

    with pytest.raises(ExceptionGroup) as exc_info:
        await ref.aio_result()

    exception_group = exc_info.value

    assert len(exception_group.exceptions) == 2

    exc_text = [e.message for e in exception_group.exceptions]

    non_retries = [
        e
        for e in exc_text
        if "This task should retry because it's not a NonRetryableException" in e
    ]

    other_errors = [e for e in exc_text if "This task should not retry" in e]

    assert len(non_retries) == 1
    assert len(other_errors) == 1

    await asyncio.sleep(3)

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
    assert retrying_events[0].task_id == task_to_id[should_retry_wrong_exception_type]

    """Three failed events should emit, one each for the two failing initial runs and one for the retry."""
    assert (
        len([e for e in runs.task_events if e.event_type == V1TaskEventType.FAILED])
        == 3
    )
