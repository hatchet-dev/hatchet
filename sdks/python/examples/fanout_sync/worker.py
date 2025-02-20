from typing import Any

from dotenv import load_dotenv
from pydantic import BaseModel

from hatchet_sdk import BaseWorkflow, ChildTriggerWorkflowOptions, Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


class ParentInput(BaseModel):
    n: int = 5


class ChildInput(BaseModel):
    a: str


parent = hatchet.declare_workflow(
    on_events=["parent:create"], input_validator=ParentInput
)
child = hatchet.declare_workflow(on_events=["child:create"], input_validator=ChildInput)


class SyncFanoutParent(BaseWorkflow):
    config = parent.config

    @hatchet.step(timeout="5m")
    def spawn(self, context: Context) -> dict[str, Any]:
        print("spawning child")

        n = parent.get_workflow_input(context).n

        runs = child.spawn_many(
            context,
            [
                child.construct_spawn_workflow_input(
                    input=ChildInput(a=str(i)),
                    key=f"child{i}",
                    options=ChildTriggerWorkflowOptions(
                        additional_metadata={"hello": "earth"}
                    ),
                )
                for i in range(n)
            ],
        )

        results = [r.sync_result() for r in runs]

        print(f"results {results}")

        return {"results": results}


class SyncFanoutChild(BaseWorkflow):
    config = child.config

    @hatchet.step()
    def process(self, context: Context) -> dict[str, str]:
        return {"status": "success " + context.workflow_input["a"]}


def main() -> None:
    worker = hatchet.worker("sync-fanout-worker", max_runs=40)
    worker.register_workflow(SyncFanoutParent())
    worker.register_workflow(SyncFanoutChild())
    worker.start()


if __name__ == "__main__":
    main()
