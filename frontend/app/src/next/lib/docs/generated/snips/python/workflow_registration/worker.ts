import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': '# > WorkflowRegistration\n\nfrom hatchet_sdk import Hatchet\n\nhatchet = Hatchet(debug=True)\n\nwf_one = hatchet.workflow(name=\'wf_one\')\nwf_two = hatchet.workflow(name=\'wf_two\')\nwf_three = hatchet.workflow(name=\'wf_three\')\nwf_four = hatchet.workflow(name=\'wf_four\')\nwf_five = hatchet.workflow(name=\'wf_five\')\n\n# define tasks here\n\n\ndef main() -> None:\n    # 👀 Register workflows directly when instantiating the worker\n    worker = hatchet.worker(\'test-worker\', slots=1, workflows=[wf_one, wf_two])\n\n    # 👀 Register a single workflow after instantiating the worker\n    worker.register_workflow(wf_three)\n\n    # 👀 Register multiple workflows in bulk after instantiating the worker\n    worker.register_workflows([wf_four, wf_five])\n\n    worker.start()\n\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/workflow_registration/worker.py',
  'blocks': {
    'workflowregistration': {
      'start': 2,
      'stop': 28
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
