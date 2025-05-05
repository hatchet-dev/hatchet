import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from datetime import datetime, timedelta\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n# > Create\nscheduled_run = hatchet.scheduled.create(\n    workflow_name='simple-workflow',\n    trigger_at=datetime.now() + timedelta(seconds=10),\n    input={\n        'data': 'simple-workflow-data',\n    },\n    additional_metadata={\n        'customer_id': 'customer-a',\n    },\n)\n\nid = scheduled_run.metadata.id  # the id of the scheduled run trigger\n\n# > Delete\nhatchet.scheduled.delete(scheduled_id=scheduled_run.metadata.id)\n\n# > List\nscheduled_runs = hatchet.scheduled.list()\n\n# > Get\nscheduled_run = hatchet.scheduled.get(scheduled_id=scheduled_run.metadata.id)\n",
  source: 'out/python/scheduled/programatic-sync.py',
  blocks: {
    create: {
      start: 8,
      stop: 19,
    },
    delete: {
      start: 22,
      stop: 22,
    },
    list: {
      start: 25,
      stop: 25,
    },
    get: {
      start: 28,
      stop: 28,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
