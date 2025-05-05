import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.blocked_async.worker import blocked_worker_workflow\nfrom hatchet_sdk import TriggerWorkflowOptions\n\nblocked_worker_workflow.run(\n    options=TriggerWorkflowOptions(additional_metadata={\'hello\': \'moon\'}),\n)\n',
  'source': 'out/python/blocked_async/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
