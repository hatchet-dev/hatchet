import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.events.worker import EVENT_KEY, event_workflow\nfrom hatchet_sdk import Hatchet, PushEventOptions\n\nhatchet = Hatchet()\n\n# > Create a filter\nhatchet.filters.create(\n    workflow_id=event_workflow.id,\n    expression=\'input.should_skip == false\',\n    scope=\'foobarbaz\',\n    payload={\n        \'main_character\': \'Anna\',\n        \'supporting_character\': \'Stiva\',\n        \'location\': \'Moscow\',\n    },\n)\n\n# > Skip a run\nhatchet.event.push(\n    event_key=EVENT_KEY,\n    payload={\n        \'should_skip\': True,\n    },\n    options=PushEventOptions(\n        scope=\'foobarbaz\',\n    ),\n)\n\n# > Trigger a run\nhatchet.event.push(\n    event_key=EVENT_KEY,\n    payload={\n        \'should_skip\': True,\n    },\n    options=PushEventOptions(\n        scope=\'foobarbaz\',\n    ),\n)\n',
  'source': 'out/python/events/filter.py',
  'blocks': {
    'create_a_filter': {
      'start': 7,
      'stop': 16
    },
    'skip_a_run': {
      'start': 19,
      'stop': 27
    },
    'trigger_a_run': {
      'start': 30,
      'stop': 38
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
