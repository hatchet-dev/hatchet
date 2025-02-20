from hatchet_sdk import BaseWorkflow, Context, Hatchet, WorkerLabelComparator
from hatchet_sdk.labels import DesiredWorkerLabel

hatchet = Hatchet(debug=True)

wf = hatchet.declare_workflow(on_events=["affinity:run"])


class AffinityWorkflow(BaseWorkflow):
    config = wf.config

    @hatchet.step(
        desired_worker_labels={
            "model": DesiredWorkerLabel(value="fancy-ai-model-v2", weight=10),
            "memory": DesiredWorkerLabel(
                value=256,
                required=True,
                comparator=WorkerLabelComparator.LESS_THAN,
            ),
        },
    )
    async def step(self, context: Context) -> dict[str, str | None]:
        if context.worker.labels().get("model") != "fancy-ai-model-v2":
            context.worker.upsert_labels({"model": "unset"})
            # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
            context.worker.upsert_labels({"model": "fancy-ai-model-v2"})

        return {"worker": context.worker.id()}


def main() -> None:
    worker = hatchet.worker(
        "affinity-worker",
        max_runs=10,
        labels={
            "model": "fancy-ai-model-v2",
            "memory": 512,
        },
    )
    worker.register_workflow(AffinityWorkflow())
    worker.start()


if __name__ == "__main__":
    main()
