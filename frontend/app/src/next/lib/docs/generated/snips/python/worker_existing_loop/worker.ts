import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\nfrom contextlib import suppress\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet\n\nhatchet = Hatchet(debug=True)\n\nexisting_loop_worker = hatchet.workflow(name="WorkerExistingLoopWorkflow")\n\n\n@existing_loop_worker.task()\nasync def task(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("started")\n    await asyncio.sleep(10)\n    print("finished")\n    return {"result": "returned result"}\n\n\nasync def async_main() -> None:\n    worker = None\n    try:\n        worker = hatchet.worker(\n            "test-worker", slots=1, workflows=[existing_loop_worker]\n        )\n        worker.start()\n\n        ref = existing_loop_worker.run_no_wait()\n        print(await ref.aio_result())\n        while True:\n            await asyncio.sleep(1)\n    finally:\n        if worker:\n            await worker.exit_gracefully()\n\n\ndef main() -> None:\n    with suppress(KeyboardInterrupt):\n        asyncio.run(async_main())\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/worker_existing_loop/worker.py',
  blocks: {},
  highlights: {},
};

export default snippet;
