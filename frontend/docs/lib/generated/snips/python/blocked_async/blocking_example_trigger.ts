import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# > Trigger\nimport time\n\nfrom examples.blocked_async.blocking_example_worker import (\n    blocking,\n    non_blocking_async,\n    non_blocking_sync,\n)\n\nnon_blocking_sync.run_no_wait()\nnon_blocking_async.run_no_wait()\n\ntime.sleep(1)\n\nblocking.run_no_wait()\n\ntime.sleep(1)\n\nnon_blocking_sync.run_no_wait()\n\n",
  "source": "out/python/blocked_async/blocking_example_trigger.py",
  "blocks": {
    "trigger": {
      "start": 2,
      "stop": 20
    }
  },
  "highlights": {}
};

export default snippet;
