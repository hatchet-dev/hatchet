import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import time\n\nfrom examples.durable.worker import (\n    EVENT_KEY,\n    SLEEP_TIME,\n    durable_workflow,\n    ephemeral_workflow,\n    hatchet,\n)\n\ndurable_workflow.run_no_wait()\nephemeral_workflow.run_no_wait()\n\nprint(\"Sleeping\")\ntime.sleep(SLEEP_TIME + 2)\n\nprint(\"Pushing event\")\nhatchet.event.push(EVENT_KEY, {})\n",
  "source": "out/python/durable/trigger.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
