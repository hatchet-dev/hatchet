import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from examples.simple.worker import hatchet\nfrom hatchet_sdk import PushEventOptions\n\nhatchet.event.push(\n    'test:event',\n    {'key': 'value', 'group': 'shouldSkip'},\n    options=PushEventOptions(\n        additional_metadata={'foo': 'bar'},\n    ),\n)\n\nhatchet.event.push(\n    'test:event',\n    {'key': 'value', 'group': 'shouldNotSkip'},\n    options=PushEventOptions(\n        additional_metadata={'foo': 'bar'},\n    ),\n)\n",
  source: 'out/python/simple/trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
