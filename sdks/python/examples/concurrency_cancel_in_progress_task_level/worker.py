import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


class WorkflowInput(BaseModel):
    group: str


# The concurrency is declared on the TASK (step-level), not on the workflow. Step-level concurrency
# is the only path served by the in-memory concurrency index, so this exercises that code path
# rather than the legacy workflow-level DB path.
concurrency_cancel_in_progress_task_level_workflow = hatchet.workflow(
    name="ConcurrencyCancelInProgressTaskLevel",
    input_validator=WorkflowInput,
)


# > Task-Level Cancel In Progress
@concurrency_cancel_in_progress_task_level_workflow.task(
    concurrency=[
        ConcurrencyExpression(
            expression="input.group",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
        )
    ],
)
async def task(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(50):
        await asyncio.sleep(0.10)


# !!


def main() -> None:
    worker = hatchet.worker(
        "concurrency-cancel-in-progress-task-level-worker",
        workflows=[concurrency_cancel_in_progress_task_level_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
