from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.clients.admin import TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 100


class ChildInput(BaseModel):
    a: str


class ChildTaskOutput(BaseModel):
    status: str


class ChildOutput(BaseModel):
    process: ChildTaskOutput
    process2: ChildTaskOutput


class ParentOutput(BaseModel):
    children: list[ChildOutput]


bulk_parent_wf = hatchet.workflow(name="BulkFanoutParent", input_validator=ParentInput)
bulk_child_wf = hatchet.workflow(
    name="BulkFanoutChild", input_validator=ChildInput, output_validator=ChildOutput
)


# ❓ BulkFanoutParent
@bulk_parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> ParentOutput:
    # 👀 Create each workflow run to spawn
    child_workflow_runs = [
        bulk_child_wf.create_bulk_run_item(
            input=ChildInput(a=str(i)),
            key=f"child{i}",
            options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
        )
        for i in range(input.n)
    ]

    # 👀 Run workflows in bulk to improve performance
    spawn_results = await bulk_child_wf.aio_run_many(child_workflow_runs)

    return ParentOutput(children=spawn_results)


# ‼️


@bulk_child_wf.task()
def process(input: ChildInput, ctx: Context) -> ChildTaskOutput:
    print(f"child process {input.a}")
    ctx.put_stream("child 1...")
    return ChildTaskOutput(status="success " + input.a)


@bulk_child_wf.task()
def process2(input: ChildInput, ctx: Context) -> ChildTaskOutput:
    print("child process2")
    ctx.put_stream("child 2...")
    return ChildTaskOutput(status="success " + input.a)


def main() -> None:
    worker = hatchet.worker(
        "fanout-worker", slots=40, workflows=[bulk_parent_wf, bulk_child_wf]
    )
    worker.start()


if __name__ == "__main__":
    main()
