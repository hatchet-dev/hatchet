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


concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow = (
    hatchet.workflow(
        name="ConcurrencyCancelQueuedExceptOldestWithParentConcurrency",
        input_validator=WorkflowInput,
        concurrency=ConcurrencyExpression(
            expression="input.group",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_QUEUED_EXCEPT_OLDEST,
        ),
    )
)


# > Task-Level Cancel Except Oldest With Parent Concurrency
@concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow.task()
async def task(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(30):
        await asyncio.sleep(0.10)


@concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow.task()
async def task_2(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(30):
        await asyncio.sleep(0.10)


# !!


def main() -> None:
    worker = hatchet.worker(
        "concurrency-cancel-queued-except-oldest-with-parent-concurrency-worker",
        workflows=[
            concurrency_cancel_queued_except_oldest_with_parent_concurrency_workflow
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
