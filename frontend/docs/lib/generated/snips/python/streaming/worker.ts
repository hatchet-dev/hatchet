import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'python',
  'content': 'import asyncio\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n# > Streaming\n\nstreaming_workflow = hatchet.workflow(name=\'StreamingWorkflow\')\n\n\n@streaming_workflow.task()\nasync def step1(input: EmptyModel, ctx: Context) -> None:\n    for i in range(10):\n        await asyncio.sleep(1)\n        ctx.put_stream(f\'Processing {i}\')\n\n\ndef main() -> None:\n    worker = hatchet.worker(\'test-worker\', workflows=[streaming_workflow])\n    worker.start()\n\n\n\nif __name__ == \'__main__\':\n    main()\n',
  'source': 'out/python/streaming/worker.py',
  'blocks': {
    'streaming': {
      'start': 8,
      'stop': 23
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
