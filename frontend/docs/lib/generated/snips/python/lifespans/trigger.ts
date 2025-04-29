import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.lifespans.worker import lifespan_workflow\n\nresult = lifespan_workflow.run()\n\nprint(result)\n',
  'source': 'out/python/lifespans/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
