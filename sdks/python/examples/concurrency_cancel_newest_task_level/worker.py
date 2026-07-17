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
concurrency_cancel_newest_task_level_workflow = hatchet.workflow(
    name="ConcurrencyCancelNewestTaskLevel",
    input_validator=WorkflowInput,
)


# > Task-Level Cancel Newest
@concurrency_cancel_newest_task_level_workflow.task(
    concurrency=[
        ConcurrencyExpression(
            expression="input.group",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_NEWEST,
        )
    ],
)
async def task(input: WorkflowInput, ctx: Context) -> None:
    # sleep long enough that newer runs pile up while this one runs; CANCEL_NEWEST rejects them
    # instead of preempting the running (oldest) run.
    for _ in range(50):
        await asyncio.sleep(0.10)


# !!


def main() -> None:
    worker = hatchet.worker(
        "concurrency-cancel-newest-task-level-worker",
        workflows=[concurrency_cancel_newest_task_level_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
