import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import time\n\nfrom examples.waits.worker import hatchet, task_condition_workflow\n\ntask_condition_workflow.run_no_wait()\n\ntime.sleep(5)\n\nhatchet.event.push(\'skip_on_event:skip\', {})\nhatchet.event.push(\'wait_for_event:start\', {})\n',
  'source': 'out/python/waits/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
