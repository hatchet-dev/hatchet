import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.logger.workflow import logging_workflow\n\nlogging_workflow.run()\n',
  source: 'out/python/logger/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
