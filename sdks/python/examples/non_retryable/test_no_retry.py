import pytest

from examples.non_retryable.worker import (
    non_retryable_workflow,
    should_be_retried,
    should_not_retry,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType


@pytest.mark.asyncio()
async def test_no_retry(hatchet: Hatchet) -> None:
    ref = await non_retryable_workflow.aio_run_no_wait()

    with pytest.raises(Exception, match="foobar"):
        await ref.aio_result()

    runs = await hatchet.runs.aio_get(ref.workflow_run_id)

    task_name_to_metadata = {
        "should_not_retry": {
            "task": should_not_retry,
            "task_run_id": next(
                t.metadata.id
                for t in runs.tasks
                if "should_not_retry" in t.display_name
            ),
        },
        "should_be_retried": {
            "task": should_be_retried,
            "task_run_id": next(
                t.metadata.id
                for t in runs.tasks
                if "should_be_retried" in t.display_name
            ),
        },
    }

    retrying_events = [
        e for e in runs.task_events if e.event_type == V1TaskEventType.RETRYING
    ]

    """Only one task should be retrying."""
    assert len(retrying_events) == 1

    retrying_event = retrying_events[0]

    """The task id of the retrying event should be assigned to the task that is retried."""
    assert (
        retrying_event.task_id
        == task_name_to_metadata["should_be_retried"]["task_run_id"]
    )

    """Three failed events should emit, one each for the initial runs and one for the retry."""
    assert (
        len([e for e in runs.task_events if e.event_type == V1TaskEventType.FAILED])
        == 3
    )
