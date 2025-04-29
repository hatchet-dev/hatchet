import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import asyncio\n\nfrom examples.streaming.worker import streaming_workflow\n\n\nasync def main() -> None:\n    ref = await streaming_workflow.aio_run_no_wait()\n    await asyncio.sleep(1)\n\n    stream = ref.stream()\n\n    async for chunk in stream:\n        print(chunk)\n\n\nif __name__ == \'__main__\':\n    import asyncio\n\n    asyncio.run(main())\n',
  'source': 'out/python/streaming/async_stream.py',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
