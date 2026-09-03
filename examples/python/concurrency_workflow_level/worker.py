import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()

SLEEP_TIME = 2
DIGIT_MAX_RUNS = 8
NAME_MAX_RUNS = 3


# > Multiple Concurrency Keys
class WorkflowInput(BaseModel):
    name: str
    digit: str


concurrency_workflow_level_workflow = hatchet.workflow(
    name="ConcurrencyWorkflowLevel",
    input_validator=WorkflowInput,
    concurrency=[
        ConcurrencyStrategy(
            expression="input.digit",
            max_runs=DIGIT_MAX_RUNS,
            strategy="GROUP_ROUND_ROBIN",
        ),
        ConcurrencyStrategy(
            expression="input.name",
            max_runs=NAME_MAX_RUNS,
            strategy="GROUP_ROUND_ROBIN",
        ),
    ],
)


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
