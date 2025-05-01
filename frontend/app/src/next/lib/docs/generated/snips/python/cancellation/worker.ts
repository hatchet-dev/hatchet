import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "import asyncio\nimport time\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\ncancellation_workflow = hatchet.workflow(name='CancelWorkflow')\n\n\n# > Self-cancelling task\n@cancellation_workflow.task()\nasync def self_cancel(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(2)\n\n    ## Cancel the task\n    await ctx.aio_cancel()\n\n    await asyncio.sleep(10)\n\n    return {'error': 'Task should have been cancelled'}\n\n\n\n\n# > Checking exit flag\n@cancellation_workflow.task()\ndef check_flag(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    for i in range(3):\n        time.sleep(1)\n\n        # Note: Checking the status of the exit flag is mostly useful for cancelling\n        # sync tasks without needing to forcibly kill the thread they're running on.\n        if ctx.exit_flag:\n            print('Task has been cancelled')\n            raise ValueError('Task has been cancelled')\n\n    return {'error': 'Task should have been cancelled'}\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker('cancellation-worker', workflows=[cancellation_workflow])\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/cancellation/worker.py',
  blocks: {
    self_cancelling_task: {
      start: 12,
      stop: 23,
    },
    checking_exit_flag: {
      start: 27,
      stop: 40,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
