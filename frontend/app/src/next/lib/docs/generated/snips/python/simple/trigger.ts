import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from examples.simple.worker import hatchet\nfrom hatchet_sdk.clients.admin import CreateFilterRequest\nfrom hatchet_sdk.clients.events import PushEventOptions\n\nhatchet._client.admin.put_filter(\n    CreateFilterRequest(\n        workflow_id='1563ea96-3d4a-4b62-ae68-94bb11ea3bb1',\n        expression='input.shouldSkipThis == true',\n        resource_hint='foobar',\n        payload={'test': 'test'},\n    )\n)\n\nhatchet.event.push(\n    'workflow-filters:test:1',\n    {'shouldSkipThis': True},\n    PushEventOptions(\n        resource_hint='foobar',\n        additional_metadata={\n            'shouldSkipThis': True,\n        },\n    ),\n)\n\nhatchet.event.push(\n    'workflow-filters:test:1',\n    {'shouldSkipThis': False},\n    PushEventOptions(\n        resource_hint='foobar',\n        additional_metadata={\n            'shouldSkipThis': False,\n        },\n    ),\n)\n",
  source: 'out/python/simple/trigger.py',
  blocks: {},
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
