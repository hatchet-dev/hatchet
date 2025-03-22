import asyncio
from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


bulk_parent_wf = hatchet.workflow(name="BulkFanoutParent", input_validator=ParentInput)
bulk_child_wf = hatchet.workflow(name="BulkFanoutChild", input_validator=ChildInput)


@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> dict[str, list[Any]]:
    print("spawning child")

    ctx.put_stream("spawning...")
    results = []

    child_workflow_runs = [
        bulk_child_wf.create_run_workflow_config(
            input=ChildInput(a=str(i)),
            key=f"child{i}",
            options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
        )
        for i in range(input.n)
    ]

    if len(child_workflow_runs) == 0:
        return {}

    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)

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
def process(input: ChildInput, ctx: Context) -> dict[str, str]:
    print(f"child process {input.a}")
    ctx.put_stream("child 1...")
    return {"status": "success " + input.a}


@bulk_child_wf.task()
def process2(input: ChildInput, ctx: Context) -> dict[str, str]:
    print("child process2")
    ctx.put_stream("child 2...")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", slots=40, workflows=[bulk_parent_wf, bulk_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
