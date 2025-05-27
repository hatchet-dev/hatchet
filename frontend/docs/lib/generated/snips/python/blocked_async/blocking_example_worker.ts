import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "# > Worker\nimport asyncio\nimport time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet()\n\nSLEEP_TIME = 6\n\n\n@hatchet.task()\nasync def non_blocking_async(input: EmptyModel, ctx: Context) -> None:\n    for i in range(SLEEP_TIME):\n        print(\"Non blocking async\", i)\n        await asyncio.sleep(1)\n\n\n@hatchet.task()\ndef non_blocking_sync(input: EmptyModel, ctx: Context) -> None:\n    for i in range(SLEEP_TIME):\n        print(\"Non blocking sync\", i)\n        time.sleep(1)\n\n\n@hatchet.task()\nasync def blocking(input: EmptyModel, ctx: Context) -> None:\n    for i in range(SLEEP_TIME):\n        print(\"Blocking\", i)\n        time.sleep(1)\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        \"test-worker\", workflows=[non_blocking_async, non_blocking_sync, blocking]\n    )\n    worker.start()\n\n\nif __name__ == \"__main__\":\n    main()\n",
  "source": "out/python/blocked_async/blocking_example_worker.py",
  "blocks": {
    "worker": {
      "start": 2,
      "stop": 32
    }
  },
  "highlights": {}
};

export default snippet;
