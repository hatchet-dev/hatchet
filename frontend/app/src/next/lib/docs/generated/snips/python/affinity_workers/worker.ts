import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabelComparator\nfrom hatchet_sdk.labels import DesiredWorkerLabel\n\nhatchet = Hatchet(debug=True)\n\n\n# > AffinityWorkflow\n\naffinity_worker_workflow = hatchet.workflow(name="AffinityWorkflow")\n\n\n@affinity_worker_workflow.task(\n    desired_worker_labels={\n        "model": DesiredWorkerLabel(value="fancy-ai-model-v2", weight=10),\n        "memory": DesiredWorkerLabel(\n            value=256,\n            required=True,\n            comparator=WorkerLabelComparator.LESS_THAN,\n        ),\n    },\n)\n\n\n\n# > AffinityTask\nasync def step(input: EmptyModel, ctx: Context) -> dict[str, str | None]:\n    if ctx.worker.labels().get("model") != "fancy-ai-model-v2":\n        ctx.worker.upsert_labels({"model": "unset"})\n        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL\n        ctx.worker.upsert_labels({"model": "fancy-ai-model-v2"})\n\n    return {"worker": ctx.worker.id()}\n\n\n\n\ndef main() -> None:\n\n    # > AffinityWorker\n    worker = hatchet.worker(\n        "affinity-worker",\n        slots=10,\n        labels={\n            "model": "fancy-ai-model-v2",\n            "memory": 512,\n        },\n        workflows=[affinity_worker_workflow],\n    )\n    worker.start()\n\n\n\nif __name__ == "__main__":\n    main()\n',
  source: 'out/python/affinity_workers/worker.py',
  blocks: {
    affinityworkflow: {
      start: 8,
      stop: 22,
    },
    affinitytask: {
      start: 26,
      stop: 34,
    },
    affinityworker: {
      start: 40,
      stop: 51,
    },
  },
  highlights: {},
};

export default snippet;
