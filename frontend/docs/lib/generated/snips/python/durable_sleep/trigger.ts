import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from examples.durable_sleep.worker import durable_sleep_task\n\ndurable_sleep_task.run_no_wait()\n',
  'source': 'out/python/durable_sleep/trigger.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
