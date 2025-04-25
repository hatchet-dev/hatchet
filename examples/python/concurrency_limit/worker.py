import time
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet(debug=True)


# ❓ Workflow
class WorkflowInput(BaseModel):
    run: int
    group_key: str


concurrency_limit_workflow = hatchet.workflow(
    name="ConcurrencyDemoWorkflow",
    concurrency=ConcurrencyExpression(
        expression="input.group_key",
        max_runs=5,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
    input_validator=WorkflowInput,
)

# ‼️


@concurrency_limit_workflow.task()
def step1(input: WorkflowInput, ctx: Context) -> dict[str, Any]:
    time.sleep(3)
    print("executed step1")
    return {"run": input.run}


def main() -> None:
    worker = hatchet.worker(
        "concurrency-demo-worker", slots=10, workflows=[concurrency_limit_workflow]
    )

    worker.start()


if __name__ == "__main__":
    main()
