import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    "from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nsimple_workflow = hatchet.workflow(name='SimpleRetryWorkflow')\nbackoff_workflow = hatchet.workflow(name='BackoffWorkflow')\n\n\n# > Simple Step Retries\n@simple_workflow.task(retries=3)\ndef always_fail(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    raise Exception('simple task failed')\n\n\n\n\n# > Retries with Count\n@simple_workflow.task(retries=3)\ndef fail_twice(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 2:\n        raise Exception('simple task failed')\n\n    return {'status': 'success'}\n\n\n\n\n# > Retries with Backoff\n@backoff_workflow.task(\n    retries=10,\n    # 👀 Maximum number of seconds to wait between retries\n    backoff_max_seconds=10,\n    # 👀 Factor to increase the wait time between retries.\n    # This sequence will be 2s, 4s, 8s, 10s, 10s, 10s... due to the maxSeconds limit\n    backoff_factor=2.0,\n)\ndef backoff_task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    if ctx.retry_count < 3:\n        raise Exception('backoff task failed')\n\n    return {'status': 'success'}\n\n\n\n\ndef main() -> None:\n    worker = hatchet.worker('backoff-worker', slots=4, workflows=[backoff_workflow])\n    worker.start()\n\n\nif __name__ == '__main__':\n    main()\n",
  source: 'out/python/retries/worker.py',
  blocks: {
    simple_step_retries: {
      start: 10,
      stop: 14,
    },
    retries_with_count: {
      start: 18,
      stop: 25,
    },
    retries_with_backoff: {
      start: 29,
      stop: 43,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
