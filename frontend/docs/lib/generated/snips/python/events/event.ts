import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import Hatchet\n\nhatchet = Hatchet()\n\n# > Event trigger\nhatchet.event.push(\'user:create\', {})\n\n',
  'source': 'out/python/events/event.py',
  'blocks': {
    'event_trigger': {
      'start': 6,
      'stop': 6
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
