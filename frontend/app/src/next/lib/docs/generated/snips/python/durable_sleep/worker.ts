import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from datetime import timedelta\n\nfrom hatchet_sdk import DurableContext, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n# > Durable Sleep\n@hatchet.durable_task(name='DurableSleepTask')\nasync def durable_sleep_task(input: EmptyModel, ctx: DurableContext) -> None:\n    res = await ctx.aio_sleep_for(timedelta(seconds=5))\n\n    print('got result', res)\n\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker('durable-sleep-worker', workflows=[durable_sleep_task])\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/durable_sleep/worker.py',
  blocks: {
    durable_sleep: {
      start: 9,
      stop: 15,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
