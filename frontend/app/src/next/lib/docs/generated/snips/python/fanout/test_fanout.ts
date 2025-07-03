import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.fanout.worker import ParentInput, parent_wf\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_run() -> None:\n    result = await parent_wf.aio_run(ParentInput(n=2))\n\n    assert len(result["spawn"]["results"]) == 2\n',
  source: 'out/python/fanout/test_fanout.py',
  blocks: {},
  highlights: {},
};

export default snippet;
