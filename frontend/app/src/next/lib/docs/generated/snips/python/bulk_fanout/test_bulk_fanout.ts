import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import pytest\n\nfrom examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\n\n\n@pytest.mark.asyncio(loop_scope=\'session\')\nasync def test_run() -> None:\n    result = await bulk_parent_wf.aio_run(input=ParentInput(n=12))\n\n    assert len(result[\'spawn\'][\'results\']) == 12\n',
  'source': 'out/python/bulk_fanout/test_bulk_fanout.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
