import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\n\nimport pytest\n\nfrom examples.on_failure.worker import on_failure_wf\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus\n\n\n@pytest.mark.asyncio(loop_scope='session')\nasync def test_run_timeout(hatchet: Hatchet) -> None:\n    run = on_failure_wf.run_no_wait()\n    try:\n        await run.aio_result()\n\n        assert False, 'Expected workflow to timeout'\n    except Exception as e:\n        assert 'step1 failed' in str(e)\n\n    await asyncio.sleep(5)  # Wait for the on_failure job to finish\n\n    details = await hatchet.runs.aio_get(run.workflow_run_id)\n\n    assert len(details.tasks) == 2\n    assert sum(t.status == V1TaskStatus.COMPLETED for t in details.tasks) == 1\n    assert sum(t.status == V1TaskStatus.FAILED for t in details.tasks) == 1\n\n    completed_task = next(\n        t for t in details.tasks if t.status == V1TaskStatus.COMPLETED\n    )\n    failed_task = next(t for t in details.tasks if t.status == V1TaskStatus.FAILED)\n\n    assert 'on_failure' in completed_task.display_name\n    assert 'step1' in failed_task.display_name\n",
  source: 'out/python/on_failure/test_on_failure.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
