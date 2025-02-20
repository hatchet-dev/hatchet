import asyncio
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import BaseWorkflow, ChildTriggerWorkflowOptions, Context, Hatchet

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


parent_wf = hatchet.declare_workflow(
    on_events=["parent:create"], input_validator=ParentInput
)
child_wf = hatchet.declare_workflow(
    on_events=["child:create"], input_validator=ChildInput
)


class Parent(BaseWorkflow):
    config = parent_wf.config

    @hatchet.step(timeout="5m")
    async def spawn(self, context: Context) -> dict[str, Any]:
        print("spawning child")

        context.put_stream("spawning...")

        n = parent_wf.get_workflow_input(context).n

        children = await asyncio.gather(
            *[
                child_wf.aio_spawn_one(
                    ctx=context,
                    input=ChildInput(a=str(i)),
                    key=f"child{i}",
                    options=ChildTriggerWorkflowOptions(
                        additional_metadata={"hello": "earth"}
                    ),
                )
                for i in range(n)
            ]
        )

        result = await asyncio.gather(*[child.result() for child in children])
        print(f"results {result}")

        return {"results": result}


class Child(BaseWorkflow):
    config = child_wf.config

    @hatchet.step()
    def process(self, context: Context) -> dict[str, str]:
        a = child_wf.get_workflow_input(context).a
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
    worker.register_workflow(Parent())
    worker.register_workflow(Child())
    worker.start()


if __name__ == "__main__":
    main()
