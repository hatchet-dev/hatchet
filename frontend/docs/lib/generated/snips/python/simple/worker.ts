import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'from hatchet_sdk import Context, EmptyModel, Hatchet\nimport time\nimport asyncio\n\nhatchet = Hatchet(debug=True)\n\n\n@hatchet.task()\nasync def simple(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(60):\n        print(f\'blocking task {i}\')\n        time.sleep(1)\n\n@hatchet.task()\nasync def non_blocking(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(60):\n        print(f\'non-blocking task {i}\')\n        await asyncio.sleep(1)\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'test-worker\', workflows=[simple, non_blocking])\n    worker.start()\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/simple/worker.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
