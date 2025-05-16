import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.events.worker import EVENT_KEY, event_workflow, hatchet\nfrom hatchet_sdk.clients.events import PushEventOptions\n\nSCOPE = \'foobar\'\nhatchet.filters.create(\n    workflow_id=event_workflow.id,\n    expression=\'input.should_skip == false\',\n    scope=SCOPE,\n    payload={\'test\': \'test\'},\n)\n\nhatchet.event.push(\n    EVENT_KEY,\n    {\'should_skip\': True},\n    PushEventOptions(\n        ## If no scope is provided, all events pushed will trigger workflows\n        ## (i.e. the filter will not apply)\n        scope=SCOPE,\n        additional_metadata={\n            \'should_skip\': True,\n        },\n    ),\n)\n\nhatchet.event.push(\n    EVENT_KEY,\n    {\'should_skip\': False},\n    PushEventOptions(\n        ## If no scope is provided, all events pushed will trigger workflows\n        ## (i.e. the filter will not apply)\n        scope=SCOPE,\n        additional_metadata={\n            \'should_skip\': False,\n        },\n    ),\n)\n',
  'source': 'out/python/events/event.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
