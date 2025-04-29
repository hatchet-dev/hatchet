import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.on_success.worker import on_success_workflow\n\non_success_workflow.run_no_wait()\n',
  'source': 'out/python/on_success/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
