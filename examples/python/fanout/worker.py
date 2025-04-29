from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


# > FanoutParent
class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


parent_wf = hatchet.workflow(name="FanoutParent", input_validator=ParentInput)
child_wf = hatchet.workflow(name="FanoutChild", input_validator=ChildInput)


@parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> dict[str, Any]:
    print("spawning child")

    result = await child_wf.aio_run_many(
        [
            child_wf.create_bulk_run_item(
                input=ChildInput(a=str(i)),
                options=TriggerWorkflowOptions(
                    additional_metadata={"hello": "earth"}, key=f"child{i}"
                ),
            )
            for i in range(input.n)
        ]
    )

    print(f"results {result}")

    return {"results": result}




# > FanoutChild
@child_wf.task()
def process(input: ChildInput, ctx: Context) -> dict[str, str]:
    print(f"child process {input.a}")
    return {"status": input.a}


@child_wf.task(parents=[process])
def process2(input: ChildInput, ctx: Context) -> dict[str, str]:
    process_output = ctx.task_output(process)
    a = process_output["status"]

    return {"status2": a + "2"}



child_wf.create_bulk_run_item()


def main() -> None:
    worker = hatchet.worker("fanout-worker", slots=40, workflows=[parent_wf, child_wf])
    worker.start()


if __name__ == "__main__":
    main()
