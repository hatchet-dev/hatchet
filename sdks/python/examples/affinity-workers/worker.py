from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabelComparator
from hatchet_sdk.labels import DesiredWorkerLabel

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(name="AffinityWorkflow", on_events=["affinity:run"])


@wf.task(
    desired_worker_labels={
        "model": DesiredWorkerLabel(value="fancy-ai-model-v2", weight=10),
        "memory": DesiredWorkerLabel(
            value=256,
            required=True,
            comparator=WorkerLabelComparator.LESS_THAN,
        ),
    },
)
async def step(input: EmptyModel, context: Context) -> dict[str, str | None]:
    if context.worker.labels().get("model") != "fancy-ai-model-v2":
        context.worker.upsert_labels({"model": "unset"})
        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
        context.worker.upsert_labels({"model": "fancy-ai-model-v2"})

    return {"worker": context.worker.id()}


def main() -> None:
    worker = hatchet.worker(
        "affinity-worker",
        slots=10,
        labels={
            "model": "fancy-ai-model-v2",
            "memory": 512,
        },
        workflows=[wf],
    )
    worker.start()


if __name__ == "__main__":
    main()
