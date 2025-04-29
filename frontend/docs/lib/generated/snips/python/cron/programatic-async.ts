import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from pydantic import BaseModel\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n\nclass DynamicCronInput(BaseModel):\n    name: str\n\n\nasync def create_cron() -> None:\n    dynamic_cron_workflow = hatchet.workflow(\n        name=\'CronWorkflow\', input_validator=DynamicCronInput\n    )\n\n    # > Create\n    cron_trigger = await dynamic_cron_workflow.aio_create_cron(\n        cron_name=\'customer-a-daily-report\',\n        expression=\'0 12 * * *\',\n        input=DynamicCronInput(name=\'John Doe\'),\n        additional_metadata={\n            \'customer_id\': \'customer-a\',\n        },\n    )\n\n    cron_trigger.metadata.id  # the id of the cron trigger\n\n    # > List\n    await hatchet.cron.aio_list()\n\n    # > Get\n    cron_trigger = await hatchet.cron.aio_get(cron_id=cron_trigger.metadata.id)\n\n    # > Delete\n    await hatchet.cron.aio_delete(cron_id=cron_trigger.metadata.id)\n',
  'source': 'out/python/cron/programatic-async.py',
  'blocks': {
    'create': {
      'start': 18,
      'stop': 27
    },
    'list': {
      'start': 30,
      'stop': 30
    },
    'get': {
      'start': 33,
      'stop': 33
    },
    'delete': {
      'start': 36,
      'stop': 36
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
