import asyncio
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


parent_wf = hatchet.workflow(name="FanoutParent", input_validator=ParentInput)
child_wf = hatchet.workflow(name="FanoutChild", input_validator=ChildInput)


@parent_wf.task(timeout="5m")
async def spawn(input: ParentInput, ctx: Context) -> dict[str, Any]:
    print("spawning child")

    children = await child_wf.aio_run_many(
        [
            child_wf.create_run_workflow_config(
                input=ChildInput(a=str(i)),
                options=TriggerWorkflowOptions(
                    additional_metadata={"hello": "earth"}, key=f"child{i}"
                ),
            )
            for i in range(input.n)
        ]
    )

    result = await asyncio.gather(*[child.aio_result() for child in children])
    print(f"results {result}")

    return {"results": result}


@child_wf.task()
def process(input: ChildInput, ctx: Context) -> dict[str, str]:
    a = child_wf.get_workflow_input(ctx).a
    print(f"child process {a}")
    return {"status": "success " + a}


@child_wf.task()
def process2(input: ChildInput, ctx: Context) -> dict[str, str]:
    print("child process2")
    return {"status2": "success"}


def main() -> None:
    worker = hatchet.worker("fanout-worker", slots=40, workflows=[parent_wf, child_wf])
    worker.start()


if __name__ == "__main__":
    main()
