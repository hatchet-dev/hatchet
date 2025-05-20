import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import asyncio\nfrom datetime import timedelta\nfrom typing import Any\n\nfrom hatchet_sdk import Context, EmptyModel, Hatchet, TriggerWorkflowOptions\nfrom hatchet_sdk.clients.admin import DedupeViolationErr\n\nhatchet = Hatchet(debug=True)\n\ndedupe_parent_wf = hatchet.workflow(name="DedupeParent")\ndedupe_child_wf = hatchet.workflow(name="DedupeChild")\n\n\n@dedupe_parent_wf.task(execution_timeout=timedelta(minutes=1))\nasync def spawn(input: EmptyModel, ctx: Context) -> dict[str, list[Any]]:\n    print("spawning child")\n\n    results = []\n\n    for i in range(2):\n        try:\n            results.append(\n                (\n                    dedupe_child_wf.aio_run(\n                        options=TriggerWorkflowOptions(\n                            additional_metadata={"dedupe": "test"}, key=f"child{i}"\n                        ),\n                    )\n                )\n            )\n        except DedupeViolationErr as e:\n            print(f"dedupe violation {e}")\n            continue\n\n    result = await asyncio.gather(*results)\n    print(f"results {result}")\n\n    return {"results": result}\n\n\n@dedupe_child_wf.task()\nasync def process(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    await asyncio.sleep(3)\n\n    print("child process")\n    return {"status": "success"}\n\n\n@dedupe_child_wf.task()\nasync def process2(input: EmptyModel, ctx: Context) -> dict[str, str]:\n    print("child process2")\n    return {"status2": "success"}\n\n\ndef main() -> None:\n    worker = hatchet.worker(\n        "fanout-worker", slots=100, workflows=[dedupe_parent_wf, dedupe_child_wf]\n    )\n    worker.start()\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/dedupe/worker.py',
  blocks: {},
  highlights: {},
};

export default snippet;
