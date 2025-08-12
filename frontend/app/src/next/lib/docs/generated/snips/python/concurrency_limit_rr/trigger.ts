import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.concurrency_limit_rr.worker import (\n    WorkflowInput,\n    concurrency_limit_rr_workflow,\n)\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\nfor i in range(200):\n    group = "0"\n\n    if i % 2 == 0:\n        group = "1"\n\n    concurrency_limit_rr_workflow.run(WorkflowInput(group=group))\n',
  source: 'out/python/concurrency_limit_rr/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
