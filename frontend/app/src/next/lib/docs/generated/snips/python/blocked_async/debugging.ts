import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    '# > Functions\nimport asyncio\nimport time\n\nsleep_time = 3\n\nasync def blocking() -> None:\n    for i in range(sleep_time):\n        print("Blocking", i)\n        time.sleep(1)\n\nasync def non_blocking(task_id: str = "Non-blocking") -> None:\n    for i in range(sleep_time):\n        print(task_id, i)\n        await asyncio.sleep(1)\n\n# > Blocked\nasync def blocked() -> None:\n    loop = asyncio.get_event_loop()\n\n    await asyncio.gather(*[\n        loop.create_task(blocking()),\n        loop.create_task(non_blocking()),\n    ])\n\n# > Unblocked\nasync def working() -> None:\n    loop = asyncio.get_event_loop()\n\n    await asyncio.gather(*[\n        loop.create_task(non_blocking("A")),\n        loop.create_task(non_blocking("B")),\n    ])\n\n\nif __name__ == "__main__":\n    asyncio.run(blocked())\n    asyncio.run(working())\n',
  source: 'out/python/blocked_async/debugging.py',
  blocks: {
    functions: {
      start: 2,
      stop: 15,
    },
    blocked: {
      start: 18,
      stop: 24,
    },
    unblocked: {
      start: 27,
      stop: 33,
    },
  },
  highlights: {},
};

export default snippet;
