from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ChildInput(BaseModel):
    a: str


class ChildTaskOutput(BaseModel):
    status: str


class ChildOutput(BaseModel):
    process: ChildTaskOutput


class ParentInput(BaseModel):
    n: int = 5


class ParentOutput(BaseModel):
    children: list[ChildOutput]


sync_fanout_parent = hatchet.workflow(
    name="SyncFanoutParent", input_validator=ParentInput, output_validator=ParentOutput
)
sync_fanout_child = hatchet.workflow(
    name="SyncFanoutChild", input_validator=ChildInput, output_validator=ChildOutput
)


@sync_fanout_parent.task(execution_timeout=timedelta(minutes=5))
def spawn(input: ParentInput, ctx: Context) -> ParentOutput:
    print("spawning child")

    results = sync_fanout_child.run_many(
        [
            sync_fanout_child.create_bulk_run_item(
                input=ChildInput(a=str(i)),
                key=f"child{i}",
                options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
            )
            for i in range(input.n)
        ],
    )

    print(f"results {results}")

    return ParentOutput(children=results)


@sync_fanout_child.task()
def process(input: ChildInput, ctx: Context) -> ChildTaskOutput:
    return ChildTaskOutput(status="success " + input.a)


def main() -> None:
    worker = hatchet.worker(
        "sync-fanout-worker",
        slots=40,
        workflows=[sync_fanout_parent, sync_fanout_child],
    )
    worker.start()


if __name__ == "__main__":
    main()
