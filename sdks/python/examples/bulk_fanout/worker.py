import asyncio
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.clients.admin import ChildTriggerWorkflowOptions
from hatchet_sdk.runnables.workflow import SpawnWorkflowInput

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


bulk_parent_wf = hatchet.workflow(
    name="BulkFanoutParent", on_events=["parent:create"], input_validator=ParentInput
)
bulk_child_wf = hatchet.workflow(
    name="BulkFanoutChild", on_events=["child:create"], input_validator=ChildInput
)


@bulk_parent_wf.task(timeout="5m")
async def spawn(input: ParentInput, context: Context) -> dict[str, list[Any]]:
    print("spawning child")

    context.put_stream("spawning...")
    results = []

    child_workflow_runs = [
        SpawnWorkflowInput(
            input=ChildInput(a=str(i)),
            key=f"child{i}",
            options=ChildTriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
        )
        for i in range(input.n)
    ]

    if len(child_workflow_runs) == 0:
        return {}

    spawn_results = await bulk_child_wf.aio_spawn_many(context, child_workflow_runs)

    results = await asyncio.gather(
        *[workflowRunRef.aio_result() for workflowRunRef in spawn_results],
        return_exceptions=True,
    )

    print("finished spawning children")

    for result in results:
        if isinstance(result, Exception):
            print(f"An error occurred: {result}")
        else:
            print(result)

    return {"results": results}


@bulk_child_wf.task()
def process(input: ChildInput, context: Context) -> dict[str, str]:
    print(f"child process {input.a}")
    context.put_stream("child 1...")
    return {"status": "success " + input.a}


@bulk_child_wf.task()
def process2(input: ChildInput, context: Context) -> dict[str, str]:
    print("child process2")
    context.put_stream("child 2...")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", max_runs=40, workflows=[bulk_parent_wf, bulk_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
