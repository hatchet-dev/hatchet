import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import random\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# Create a list of events with desired distribution\nevents = [\'1\'] * 10000 + [\'0\'] * 100\nrandom.shuffle(events)\n\n# Send the shuffled events\nfor group in events:\n    hatchet.event.push(\'concurrency-test\', {\'group\': group})\n',
  'source': 'out/python/concurrency_limit_rr_load/event.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
