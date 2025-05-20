import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "from examples.non_retryable.worker import non_retryable_workflow\n\nnon_retryable_workflow.run_no_wait()\n",
  "source": "out/python/non_retryable/trigger.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
