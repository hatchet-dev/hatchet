import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\n# > Event trigger\nevent_workflow = hatchet.workflow(name=\'EventWorkflow\', on_events=[\'user:create\'])\n\n\n@event_workflow.task()\ndef task(input: EmptyModel, ctx: Context) -> None:\n    print(\'event received\')\n',
  'source': 'out/python/events/worker.py',
  'blocks': {
    'event_trigger': {
      'start': 6,
      'stop': 6
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
