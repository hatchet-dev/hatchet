import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.rate_limit.worker import rate_limit_workflow\nfrom hatchet_sdk.hatchet import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nrate_limit_workflow.run()\nrate_limit_workflow.run()\nrate_limit_workflow.run()\n',
  source: 'out/python/rate_limit/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
