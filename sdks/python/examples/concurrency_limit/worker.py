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


class WorkflowInput(BaseModel):
    run: int
    group: str


wf = hatchet.workflow(
    name="ConcurrencyDemoWorkflow",
    on_events=["concurrency-test"],
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=5,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    ),
    input_validator=WorkflowInput,
)


@wf.task()
def step1(input: WorkflowInput, context: Context) -> dict[str, Any]:
    time.sleep(3)
    print("executed step1")
    return {"run": input.run}


def main() -> None:
    worker = hatchet.worker("concurrency-demo-worker", max_runs=10, workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()
