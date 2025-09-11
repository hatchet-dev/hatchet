from hatchet_sdk import Context, EmptyModel, Hatchet, WorkerLabelComparator
from hatchet_sdk.labels import DesiredWorkerLabel

hatchet = Hatchet(debug=True)


# > AffinityWorkflow

affinity_worker_workflow = hatchet.workflow(name="AffinityWorkflow")


@affinity_worker_workflow.task(
    desired_worker_labels={
        "model": DesiredWorkerLabel(value="fancy-ai-model-v2", weight=10),
        "memory": DesiredWorkerLabel(
            value=256,
            required=True,
            comparator=WorkerLabelComparator.LESS_THAN,
        ),
    },
)

# !!


# > AffinityTask
async def step(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    if ctx.worker.labels().get("model") != "fancy-ai-model-v2":
        ctx.worker.upsert_labels({"model": "unset"})
        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
        ctx.worker.upsert_labels({"model": "fancy-ai-model-v2"})

    return {"worker": ctx.worker.id()}


# !!


def main() -> None:

    # > AffinityWorker
    worker = hatchet.worker(
        "affinity-worker",
        slots=10,
        labels={
            "model": "fancy-ai-model-v2",
            "memory": 512,
        },
        workflows=[affinity_worker_workflow],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
