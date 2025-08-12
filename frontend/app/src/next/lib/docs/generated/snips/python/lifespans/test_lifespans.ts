import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.lifespans.simple import Lifespan, lifespan_task\n\n\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_lifespans() -> None:\n    result = await lifespan_task.aio_run()\n\n    assert isinstance(result, Lifespan)\n    assert result.pi == 3.14\n    assert result.foo == "bar"\n',
  source: 'out/python/lifespans/test_lifespans.py',
  blocks: {},
  highlights: {},
};

export default snippet;
