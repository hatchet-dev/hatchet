from datetime import timedelta
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 5


class ChildInput(BaseModel):
    a: str


sync_fanout_parent = hatchet.workflow(
    name="SyncFanoutParent", input_validator=ParentInput
)
sync_fanout_child = hatchet.workflow(name="SyncFanoutChild", input_validator=ChildInput)


@sync_fanout_parent.task(execution_timeout=timedelta(minutes=5))
def spawn(input: ParentInput, ctx: Context) -> dict[str, list[dict[str, Any]]]:
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

    return {"results": results}


@sync_fanout_child.task()
def process(input: ChildInput, ctx: Context) -> dict[str, str]:
    return {"status": "success " + input.a}


@sync_fanout_child.task(parents=[process])
def process2(input: ChildInput, ctx: Context) -> dict[str, str]:
    process_output = ctx.task_output(process)
    a = process_output["status"]

    return {"status2": a + "2"}


def main() -> None:
    worker = hatchet.worker(
        "sync-fanout-worker",
        slots=40,
        workflows=[sync_fanout_parent, sync_fanout_child],
    )
    worker.start()


if __name__ == "__main__":
    main()
