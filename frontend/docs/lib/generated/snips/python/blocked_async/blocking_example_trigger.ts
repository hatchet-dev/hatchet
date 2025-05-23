import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# > Trigger\n\nfrom examples.blocked_async.blocking_example_worker import non_blocking_async, non_blocking_sync, blocking\nimport time\n\nnon_blocking_sync.run_no_wait()\nnon_blocking_async.run_no_wait()\n\ntime.sleep(1)\n\nblocking.run_no_wait()\n\ntime.sleep(1)\n\nnon_blocking_sync.run_no_wait()\n",
  "source": "out/python/blocked_async/blocking_example_trigger.py",
  "blocks": {
    "trigger": {
      "start": 2,
      "stop": 16
    }
  },
  "highlights": {}
};

export default snippet;
