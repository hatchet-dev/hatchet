import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\n\nimport pytest\n\nfrom examples.cancellation.worker import cancellation_workflow\nfrom hatchet_sdk import Hatchet\nfrom hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_cancellation(hatchet: Hatchet) -> None:\n    ref = await cancellation_workflow.aio_run_no_wait()\n\n    """Sleep for a long time since we only need cancellation to happen _eventually_"""\n    await asyncio.sleep(10)\n\n    for i in range(30):\n        run = await hatchet.runs.aio_get(ref.workflow_run_id)\n\n        if run.run.status == V1TaskStatus.RUNNING:\n            await asyncio.sleep(1)\n            continue\n\n        assert run.run.status == V1TaskStatus.CANCELLED\n        assert not run.run.output\n\n        break\n    else:\n        assert False, "Workflow run did not cancel in time"\n',
  source: 'out/python/cancellation/test_cancellation.py',
  blocks: {},
  highlights: {},
};

export default snippet;
