import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from uuid import uuid4\n\nimport pytest\n\nfrom examples.fanout_sync.worker import ParentInput, sync_fanout_parent\nfrom hatchet_sdk import Hatchet, TriggerWorkflowOptions\n\n\ndef test_run() -> None:\n    N = 2\n\n    result = sync_fanout_parent.run(ParentInput(n=N))\n\n    assert len(result["spawn"]["results"]) == N\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_additional_metadata_propagation_sync(hatchet: Hatchet) -> None:\n    test_run_id = uuid4().hex\n\n    ref = await sync_fanout_parent.aio_run_no_wait(\n        ParentInput(n=2),\n        options=TriggerWorkflowOptions(\n            additional_metadata={"test_run_id": test_run_id}\n        ),\n    )\n\n    await ref.aio_result()\n\n    runs = await hatchet.runs.aio_list(\n        parent_task_external_id=ref.workflow_run_id,\n        additional_metadata={"test_run_id": test_run_id},\n    )\n\n    assert runs.rows\n\n    """Assert that the additional metadata is propagated to the child runs."""\n    for run in runs.rows:\n        assert run.additional_metadata\n        assert run.additional_metadata["test_run_id"] == test_run_id\n\n        assert run.children\n        for child in run.children:\n            assert child.additional_metadata\n            assert child.additional_metadata["test_run_id"] == test_run_id\n',
  source: 'out/python/fanout_sync/test_fanout_sync.py',
  blocks: {},
  highlights: {},
};

export default snippet;
