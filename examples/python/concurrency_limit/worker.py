import time
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


# > Workflow
class WorkflowInput(BaseModel):
    run: int
    group_key: str


concurrency_limit_workflow = hatchet.workflow(
    name="ConcurrencyDemoWorkflow",
    concurrency=ConcurrencyStrategy(
        expression="input.group_key",
        max_runs=5,
        strategy="CANCEL_IN_PROGRESS",
    ),
    input_validator=WorkflowInput,
)



@concurrency_limit_workflow.task()
def step1(input: WorkflowInput, ctx: Context) -> dict[str, Any]:
    time.sleep(3)
    print("executed step1")
    return {"run": input.run}


# > Slots
def main() -> None:
    worker = hatchet.worker(
        "concurrency-demo-worker", slots=10, workflows=[concurrency_limit_workflow]
    )

    worker.start()




if __name__ == "__main__":
    main()
