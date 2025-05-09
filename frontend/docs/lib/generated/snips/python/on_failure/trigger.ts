import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.on_failure.worker import on_failure_wf_with_details\n\non_failure_wf_with_details.run_no_wait()\n',
  'source': 'out/python/on_failure/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
