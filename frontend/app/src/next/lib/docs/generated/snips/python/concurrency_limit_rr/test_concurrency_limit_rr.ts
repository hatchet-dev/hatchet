import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\n\nimport pytest\n\nfrom examples.concurrency_limit_rr.worker import concurrency_limit_rr_workflow\nfrom hatchet_sdk.workflow_run import WorkflowRunRef\n\n\n@pytest.mark.skip(reason=\'The timing for this test is not reliable\')\n@pytest.mark.asyncio(loop_scope=\'session\')\nasync def test_run() -> None:\n    num_groups = 2\n    runs: list[WorkflowRunRef] = []\n\n    # Start all runs\n    for i in range(1, num_groups + 1):\n        run = concurrency_limit_rr_workflow.run_no_wait()\n        runs.append(run)\n        run = concurrency_limit_rr_workflow.run_no_wait()\n        runs.append(run)\n\n    # Wait for all results\n    successful_runs = []\n    cancelled_runs = []\n\n    start_time = time.time()\n\n    # Process each run individually\n    for i, run in enumerate(runs, start=1):\n        try:\n            result = await run.aio_result()\n            successful_runs.append((i, result))\n        except Exception as e:\n            if \'CANCELLED_BY_CONCURRENCY_LIMIT\' in str(e):\n                cancelled_runs.append((i, str(e)))\n            else:\n                raise  # Re-raise if it\'s an unexpected error\n\n    end_time = time.time()\n    total_time = end_time - start_time\n\n    # Check that we have the correct number of successful and cancelled runs\n    assert (\n        len(successful_runs) == 4\n    ), f\'Expected 4 successful runs, got {len(successful_runs)}\'\n    assert (\n        len(cancelled_runs) == 0\n    ), f\'Expected 0 cancelled run, got {len(cancelled_runs)}\'\n\n    # Check that the total time is close to 2 seconds\n    assert (\n        3.8 <= total_time <= 7\n    ), f\'Expected runtime to be about 4 seconds, but it took {total_time:.2f} seconds\'\n\n    print(f\'Total execution time: {total_time:.2f} seconds\')\n',
  'source': 'out/python/concurrency_limit_rr/test_concurrency_limit_rr.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
