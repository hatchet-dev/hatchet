import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from pydantic import BaseModel\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n\nclass DynamicCronInput(BaseModel):\n    name: str\n\n\ndynamic_cron_workflow = hatchet.workflow(\n    name="CronWorkflow", input_validator=DynamicCronInput\n)\n\n# > Create\ncron_trigger = dynamic_cron_workflow.create_cron(\n    cron_name="customer-a-daily-report",\n    expression="0 12 * * *",\n    input=DynamicCronInput(name="John Doe"),\n    additional_metadata={\n        "customer_id": "customer-a",\n    },\n)\n\n\nid = cron_trigger.metadata.id  # the id of the cron trigger\n\n# > List\ncron_triggers = hatchet.cron.list()\n\n# > Get\ncron_trigger = hatchet.cron.get(cron_id=cron_trigger.metadata.id)\n\n# > Delete\nhatchet.cron.delete(cron_id=cron_trigger.metadata.id)\n',
  source: 'out/python/cron/programatic-sync.py',
  blocks: {
    create: {
      start: 17,
      stop: 27,
    },
    list: {
      start: 30,
      stop: 30,
    },
    get: {
      start: 33,
      stop: 33,
    },
    delete: {
      start: 36,
      stop: 36,
    },
  },
  highlights: {},
};

export default snippet;
