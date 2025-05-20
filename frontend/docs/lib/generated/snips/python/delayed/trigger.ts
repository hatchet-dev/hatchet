import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from examples.delayed.worker import PrinterInput, print_schedule_wf\n\nprint_schedule_wf.run(PrinterInput(message=\"test\"))\n",
  "source": "out/python/delayed/trigger.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
