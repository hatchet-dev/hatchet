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


# The workflow declares its own ("parent") concurrency in addition to the task-level
# ("child") concurrency below, to make sure CANCEL_EXCEPT_NEWEST still works correctly
# at the task level when a separate workflow-level concurrency scope is also in play.
# The parent's max_runs is set high enough that it never actually binds during the test,
# isolating the assertions to the task-level strategy.
concurrency_cancel_except_newest_with_parent_concurrency_workflow = hatchet.workflow(
    name="ConcurrencyCancelExceptNewestWithParentConcurrency",
    input_validator=WorkflowInput,
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=50,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


# > Task-Level Cancel Except Newest With Parent Concurrency
@concurrency_cancel_except_newest_with_parent_concurrency_workflow.task(
    concurrency=[
        ConcurrencyExpression(
            expression="input.group",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_EXCEPT_NEWEST,
        )
    ],
)
async def task(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(30):
        await asyncio.sleep(0.10)


# !!


def main() -> None:
    worker = hatchet.worker(
        "concurrency-cancel-except-newest-with-parent-concurrency-worker",
        workflows=[concurrency_cancel_except_newest_with_parent_concurrency_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
