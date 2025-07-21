import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\nfrom uuid import uuid4\n\nimport pytest\n\nfrom examples.fanout.worker import ParentInput, parent_wf\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run(hatchet: Hatchet) -> None:\n    ref = await parent_wf.aio_run_no_wait(\n        ParentInput(n=2),\n    )\n\n    result = await ref.aio_result()\n\n    assert len(result["spawn"]["results"]) == 2\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_additional_metadata_propagation(hatchet: Hatchet) -> None:\n    test_run_id = uuid4().hex\n\n    ref = await parent_wf.aio_run_no_wait(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(\n            additional_metadata={"test_run_id": test_run_id}\n        ),\n    )\n\n    await ref.aio_result()\n    await asyncio.sleep(1)\n\n    runs = await hatchet.runs.aio_list(\n        parent_task_external_id=ref.workflow_run_id,\n        additional_metadata={"test_run_id": test_run_id},\n    )\n\n    assert runs.rows\n\n    """Assert that the additional metadata is propagated to the child runs."""\n    for run in runs.rows:\n        assert run.additional_metadata\n        assert run.additional_metadata["test_run_id"] == test_run_id\n\n        assert run.children\n        for child in run.children:\n            assert child.additional_metadata\n            assert child.additional_metadata["test_run_id"] == test_run_id\n',
  source: 'out/python/fanout/test_fanout.py',
  blocks: {},
  highlights: {},
};

export default snippet;
