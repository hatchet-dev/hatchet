import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.dag.worker import dag_workflow\n\ndag_workflow.run()\n',
  'source': 'out/python/dag/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
