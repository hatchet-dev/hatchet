import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import pytest\n\nfrom examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow\nfrom hatchet_sdk.workflow_run import WorkflowRunRef\n\n\n@pytest.mark.asyncio(loop_scope=\'session\')\n@pytest.mark.skip(reason=\'The timing for this test is not reliable\')\nasync def test_run() -> None:\n    num_runs = 6\n    runs: list[WorkflowRunRef] = []\n\n    # Start all runs\n    for i in range(1, num_runs + 1):\n        run = concurrency_limit_workflow.run_no_wait(\n            WorkflowInput(run=i, group_key=str(i))\n        )\n        runs.append(run)\n\n    # Wait for all results\n    successful_runs = []\n    cancelled_runs = []\n\n    # Process each run individually\n    for i, run in enumerate(runs, start=1):\n        try:\n            result = await run.aio_result()\n            successful_runs.append((i, result))\n        except Exception as e:\n            if \'CANCELLED_BY_CONCURRENCY_LIMIT\' in str(e):\n                cancelled_runs.append((i, str(e)))\n            else:\n                raise  # Re-raise if it\'s an unexpected error\n\n    # Check that we have the correct number of successful and cancelled runs\n    assert (\n        len(successful_runs) == 5\n    ), f\'Expected 5 successful runs, got {len(successful_runs)}\'\n    assert (\n        len(cancelled_runs) == 1\n    ), f\'Expected 1 cancelled run, got {len(cancelled_runs)}\'\n',
  'source': 'out/python/concurrency_limit/test_concurrency_limit.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
