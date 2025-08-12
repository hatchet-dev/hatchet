import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\n\n@hatchet.task()\ndef bulk_replay_test_1(input: EmptyModel, ctx: Context) -> None:\n    print("retrying bulk replay test task", ctx.retry_count)\n    if ctx.retry_count == 0:\n        raise ValueError("This is a test error to trigger a retry.")\n\n\n@hatchet.task()\ndef bulk_replay_test_2(input: EmptyModel, ctx: Context) -> None:\n    print("retrying bulk replay test task", ctx.retry_count)\n    if ctx.retry_count == 0:\n        raise ValueError("This is a test error to trigger a retry.")\n\n\n@hatchet.task()\ndef bulk_replay_test_3(input: EmptyModel, ctx: Context) -> None:\n    print("retrying bulk replay test task", ctx.retry_count)\n    if ctx.retry_count == 0:\n        raise ValueError("This is a test error to trigger a retry.")\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "bulk-replay-test-worker",\n        workflows=[bulk_replay_test_1, bulk_replay_test_2, bulk_replay_test_3],\n    )\n\n    worker.start()\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/bulk_operations/worker.py',
  blocks: {},
  highlights: {},
};

export default snippet;
