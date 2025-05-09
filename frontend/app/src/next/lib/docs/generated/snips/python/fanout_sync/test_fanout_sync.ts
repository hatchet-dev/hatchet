import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from examples.fanout_sync.worker import ParentInput, sync_fanout_parent\n\n\ndef test_run() -> None:\n    N = 2\n\n    result = sync_fanout_parent.run(ParentInput(n=N))\n\n    assert len(result['spawn']['results']) == N\n",
  source: 'out/python/fanout_sync/test_fanout_sync.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
