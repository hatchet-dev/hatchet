import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.simple.worker import simple, non_blocking\n\nsimple.run_no_wait()\nnon_blocking.run_no_wait()\n',
  source: 'out/python/simple/trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
