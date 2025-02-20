import asyncio
from typing import Any

from dotenv import load_dotenv

from hatchet_sdk import Context, StickyStrategy, WorkerLabelComparator
from hatchet_sdk.v2.callable import DurableContext
from hatchet_sdk.v2.hatchet import Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.durable(
    sticky=StickyStrategy.HARD,
    desired_worker_labels={
        "running_workflow": {
            "value": "True",
            "required": True,
            "comparator": WorkerLabelComparator.NOT_EQUAL,
        },
    },
)
async def my_durable_func(context: DurableContext) -> dict[str, Any]:
    try:
        ref = await context.aio.spawn_workflow(
            "StickyChildWorkflow", {}, options={"sticky": True}
        )
        result = await ref.result()
    except Exception as e:
        result = str(e)

    await context.worker.async_upsert_labels({"running_workflow": "False"})
    return {"worker_result": result}


@hatchet.workflow(on_events=["sticky:child"], sticky=StickyStrategy.HARD)
class StickyChildWorkflow:
    @hatchet.step(
        desired_worker_labels={
            "running_workflow": {
                "value": "True",
                "required": True,
                "comparator": WorkerLabelComparator.NOT_EQUAL,
            },
        },
    )
    async def child(self, context: Context) -> dict[str, str | None]:
        await context.worker.async_upsert_labels({"running_workflow": "True"})

        print(f"Heavy work started on {context.worker.id()}")
        await asyncio.sleep(15)
        print(f"Finished Heavy work on {context.worker.id()}")

        return {"worker": context.worker.id()}


def main() -> None:
    worker = hatchet.worker(
        "sticky-worker",
        max_runs=10,
        labels={"running_workflow": "False"},
    )

    worker.register_workflow(StickyChildWorkflow())

    worker.start()


if __name__ == "__main__":
    main()
