import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from examples.delayed.worker import PrinterInput, print_schedule_wf\n\nprint_schedule_wf.run(PrinterInput(message="test"))\n',
  source: 'out/python/delayed/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
