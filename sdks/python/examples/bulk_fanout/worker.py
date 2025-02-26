import asyncio
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import BaseWorkflow, Context, Hatchet
from hatchet_sdk.clients.admin import ChildTriggerWorkflowOptions
from hatchet_sdk.workflow import SpawnWorkflowInput

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


bulk_parent_wf = hatchet.declare_workflow(
    on_events=["parent:create"], input_validator=ParentInput
)
bulk_child_wf = hatchet.declare_workflow(
    on_events=["child:create"], input_validator=ChildInput
)


class BulkParent(BaseWorkflow):
    config = bulk_parent_wf.config

    @hatchet.step(timeout="5m")
    async def spawn(self, context: Context) -> dict[str, list[Any]]:
        print("spawning child")

        context.put_stream("spawning...")
        results = []

        n = bulk_parent_wf.get_workflow_input(context).n

        child_workflow_runs = [
            SpawnWorkflowInput(
                input=ChildInput(a=str(i)),
                key=f"child{i}",
                options=ChildTriggerWorkflowOptions(
                    additional_metadata={"hello": "earth"}
                ),
            )
            for i in range(n)
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


class BulkChild(BaseWorkflow):
    config = bulk_child_wf.config

    @hatchet.step()
    def process(self, context: Context) -> dict[str, str]:
        a = bulk_child_wf.get_workflow_input(context).a
        print(f"child process {a}")
        context.put_stream("child 1...")
        return {"status": "success " + a}

    @hatchet.step()
    def process2(self, context: Context) -> dict[str, str]:
        print("child process2")
        context.put_stream("child 2...")
        return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker("fanout-worker", max_runs=40)
    worker.register_workflow(BulkParent())
    worker.register_workflow(BulkChild())
    worker.start()


if __name__ == "__main__":
    main()
