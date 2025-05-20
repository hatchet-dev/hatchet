import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import time\n\nfrom examples.durable_event.worker import (\n    EVENT_KEY,\n    durable_event_task,\n    durable_event_task_with_filter,\n    hatchet,\n)\n\ndurable_event_task.run_no_wait()\ndurable_event_task_with_filter.run_no_wait()\n\nprint("Sleeping")\ntime.sleep(2)\n\nprint("Pushing event")\nhatchet.event.push(\n    EVENT_KEY,\n    {\n        "user_id": "1234",\n    },\n)\n',
  source: 'out/python/durable_event/trigger.py',
  blocks: {},
  highlights: {},
};

export default snippet;
