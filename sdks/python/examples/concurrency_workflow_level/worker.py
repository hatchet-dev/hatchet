import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet(debug=True)

SLEEP_TIME = 2
DIGIT_MAX_RUNS = 8
NAME_MAX_RUNS = 3


# > Multiple Concurrency Keys
class WorkflowInput(BaseModel):
    name: str
    digit: str


concurrency_workflow_level_workflow = hatchet.workflow(
    name="ConcurrencyWorkflowManyKeys",
    input_validator=WorkflowInput,
    concurrency=[
        ConcurrencyExpression(
            expression="input.digit",
            max_runs=DIGIT_MAX_RUNS,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
        ConcurrencyExpression(
            expression="input.name",
            max_runs=NAME_MAX_RUNS,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
    ],
)
# !!


@concurrency_workflow_level_workflow.task()
async def task_1(input: WorkflowInput, ctx: Context) -> None:
    await asyncio.sleep(SLEEP_TIME)


@concurrency_workflow_level_workflow.task()
async def task_2(input: WorkflowInput, ctx: Context) -> None:
    await asyncio.sleep(SLEEP_TIME)


def main() -> None:
    worker = hatchet.worker(
        "concurrency-worker-workflow-level",
        slots=10,
        workflows=[concurrency_workflow_level_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
