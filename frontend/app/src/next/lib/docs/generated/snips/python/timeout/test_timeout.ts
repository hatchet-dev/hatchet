import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.timeout.worker import refresh_timeout_wf, timeout_wf\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_execution_timeout() -> None:\n    run = timeout_wf.run_no_wait()\n\n    with pytest.raises(\n        Exception,\n        match="(Task exceeded timeout|TIMED_OUT|Workflow run .* failed with multiple errors)",\n    ):\n        await run.aio_result()\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run_refresh_timeout() -> None:\n    result = await refresh_timeout_wf.aio_run()\n\n    assert result["refresh_task"]["status"] == "success"\n',
  source: 'out/python/timeout/test_timeout.py',
  blocks: {},
  highlights: {},
};

export default snippet;
