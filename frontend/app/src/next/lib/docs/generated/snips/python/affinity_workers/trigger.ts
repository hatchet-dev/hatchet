import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.affinity_workers.worker import affinity_worker_workflow\nfrom hatchet_sdk import TriggerWorkflowOptions\n\naffinity_worker_workflow.run(\n    options=TriggerWorkflowOptions(additional_metadata={\'hello\': \'moon\'}),\n)\n',
  'source': 'out/python/affinity_workers/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
