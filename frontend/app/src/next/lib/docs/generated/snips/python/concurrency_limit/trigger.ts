import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow\n\nconcurrency_limit_workflow.run(WorkflowInput(group_key='test', run=1))\n",
  source: 'out/python/concurrency_limit/trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
