from typing import Any

from dotenv import load_dotenv
from pydantic import BaseModel

from hatchet_sdk import ChildTriggerWorkflowOptions, Context, Hatchet
from hatchet_sdk.runnables.workflow import SpawnWorkflowInput

load_dotenv()

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

    runs = child.spawn_many(
        context,
        [
            SpawnWorkflowInput(
                input=ChildInput(a=str(i)),
                key=f"child{i}",
                options=ChildTriggerWorkflowOptions(
                    additional_metadata={"hello": "earth"}
                ),
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
    worker = hatchet.worker(
        "sync-fanout-worker", max_runs=40, workflows=[parent, child]
    )
    worker.start()


if __name__ == "__main__":
    main()
