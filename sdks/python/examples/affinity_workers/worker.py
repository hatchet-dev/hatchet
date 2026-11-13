from hatchet_sdk import Context, Hatchet, WorkerLabel, WorkerLabelComparator
from hatchet_sdk import DesiredWorkerLabel

hatchet = Hatchet()


# > AffinityWorkflow

affinity_worker_workflow = hatchet.workflow(name="AffinityWorkflow")


@affinity_worker_workflow.task(
    desired_worker_labels=[
        DesiredWorkerLabel(key="model", value="fancy-ai-model-v2", weight=10),
        DesiredWorkerLabel(
            key="memory",
            value=256,
            required=True,
            comparator="LESS_THAN",
        ),
    ],
)

# !!


# > AffinityTask
async def step(input: None, ctx: Context) -> dict[str, str | None]:
    model_labels = [label for label in ctx.worker_labels if label.key == "model"]

    if not model_labels or model_labels[0].value != "fancy-ai-model-v2":
        await ctx.aio_upsert_worker_labels([WorkerLabel(key="model", value="unset")])
        # DO WORK TO EVICT OLD MODEL / LOAD NEW MODEL
        await ctx.aio_upsert_worker_labels(
            [WorkerLabel(key="model", value="fancy-ai-model-v2")]
        )

    return {"worker": ctx.worker_id}


# !!


def main() -> None:

    # > AffinityWorker
    worker = hatchet.worker(
        "affinity-worker",
        slots=10,
        labels=[
            WorkerLabel(key="model", value="fancy-ai-model-v2"),
            WorkerLabel(key="memory", value=512),
        ],
        workflows=[affinity_worker_workflow],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
