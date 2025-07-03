import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf\nfrom hatchet_sdk import TriggerWorkflowOptions\n\nbulk_parent_wf.run(\n    ParentInput(n=999),\n    TriggerWorkflowOptions(additional_metadata={"no-dedupe": "world"}),\n)\n',
  source: 'out/python/bulk_fanout/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
