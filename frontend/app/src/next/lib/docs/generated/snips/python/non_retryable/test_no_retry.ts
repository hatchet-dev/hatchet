import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.non_retryable.worker import (\n    non_retryable_workflow,\n    should_not_retry,\n    should_not_retry_successful_task,\n    should_retry_wrong_exception_type,\n)\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_event_type import V1TaskEventType\nfrom hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails\n\n\ndef find_id(runs: V1WorkflowRunDetails, match: str) -> str:\n    return next(t.metadata.id for t in runs.tasks if match in t.display_name)\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_no_retry(hatchet: Hatchet) -> None:\n    ref = await non_retryable_workflow.aio_run_no_wait()\n\n    with pytest.raises(Exception, match="retry"):\n        await ref.aio_result()\n\n    runs = await hatchet.runs.aio_get(ref.workflow_run_id)\n    task_to_id = {\n        task: find_id(runs, task.name)\n        for task in [\n            should_not_retry_successful_task,\n            should_retry_wrong_exception_type,\n            should_not_retry,\n        ]\n    }\n\n    retrying_events = [\n        e for e in runs.task_events if e.event_type == V1TaskEventType.RETRYING\n    ]\n\n    """Only one task should be retried."""\n    assert len(retrying_events) == 1\n\n    """The task id of the retrying events should match the tasks that are retried"""\n    assert {e.task_id for e in retrying_events} == {\n        task_to_id[should_retry_wrong_exception_type],\n    }\n\n    """Three failed events should emit, one each for the two failing initial runs and one for the retry."""\n    assert (\n        len([e for e in runs.task_events if e.event_type == V1TaskEventType.FAILED])\n        == 3\n    )\n',
  source: 'out/python/non_retryable/test_no_retry.py',
  blocks: {},
  highlights: {},
};

export default snippet;
