from typing import Any

from pydantic import BaseModel

from hatchet_sdk import Context, Hatchet, TriggerWorkflowOptions

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 5


class ChildInput(BaseModel):
    a: str


parent = hatchet.workflow(
    name="SyncFanoutParent", on_events=["parent:create"], input_validator=ParentInput
)
child = hatchet.workflow(
    name="SyncFanoutChild", on_events=["child:create"], input_validator=ChildInput
)


@parent.task(timeout="5m")
def spawn(input: ParentInput, context: Context) -> dict[str, Any]:
    print("spawning child")

    runs = child.run_many(
        [
            child.create_run_workflow_config(
                input=ChildInput(a=str(i)),
                key=f"child{i}",
                options=TriggerWorkflowOptions(additional_metadata={"hello": "earth"}),
            )
            for i in range(input.n)
        ],
    )

    results = [r.result() for r in runs]

    print(f"results {results}")

    return {"results": results}


@child.task()
def process(input: ChildInput, context: Context) -> dict[str, str]:
    return {"status": "success " + input.a}


def main() -> None:
    worker = hatchet.worker("sync-fanout-worker", slots=40, workflows=[parent, child])
    worker.start()


if __name__ == "__main__":
    main()
