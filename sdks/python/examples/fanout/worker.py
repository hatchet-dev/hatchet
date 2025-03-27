from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


# ❓ FanoutParent


class ChildInput(BaseModel):
    a: str


class ChildTaskOutput(BaseModel):
    status: str


class ChildOutput(BaseModel):
    process: ChildTaskOutput


class ParentInput(BaseModel):
    n: int = 5


class ParentTaskOutput(BaseModel):
    children: list[ChildOutput]


class ParentOutput(BaseModel):
    spawn: ParentTaskOutput


parent_wf = hatchet.workflow(
    name="FanoutParent", input_validator=ParentInput, output_validator=ParentOutput
)
child_wf = hatchet.workflow(
    name="FanoutChild", input_validator=ChildInput, output_validator=ChildOutput
)


@parent_wf.task(execution_timeout=timedelta(minutes=5))
async def spawn(input: ParentInput, ctx: Context) -> ParentTaskOutput:
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

    return ParentTaskOutput(children=result)


# ‼️

# ❓ FanoutChild


@child_wf.task()
def process(input: ChildInput, ctx: Context) -> ChildTaskOutput:
    print(f"child process {input.a}")
    return ChildTaskOutput(status="success " + input.a)


@child_wf.task(parents=[process])
def process2(input: ChildInput, ctx: Context) -> ChildTaskOutput:
    process_output = ctx.task_output(process)
    a = process_output.status

    return ChildTaskOutput(status="success " + a)


# ‼️

child_wf.create_bulk_run_item()


def main() -> None:
    worker = hatchet.worker("fanout-worker", slots=40, workflows=[parent_wf, child_wf])
    worker.start()


if __name__ == "__main__":
    main()
