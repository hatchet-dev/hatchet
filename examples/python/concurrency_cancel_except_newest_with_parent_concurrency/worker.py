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


concurrency_cancel_except_newest_with_parent_concurrency_workflow = hatchet.workflow(
    name="ConcurrencyCancelExceptNewestWithParentConcurrency",
    input_validator=WorkflowInput,
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_EXCEPT_NEWEST,
    ),
)


# > Task-Level Cancel Except Newest With Parent Concurrency
@concurrency_cancel_except_newest_with_parent_concurrency_workflow.task()
async def task(input: WorkflowInput, ctx: Context) -> None:
    await asyncio.sleep(5)


@concurrency_cancel_except_newest_with_parent_concurrency_workflow.task()
async def task_2(input: WorkflowInput, ctx: Context) -> None:
    await asyncio.sleep(5)




def main() -> None:
    worker = hatchet.worker(
        "concurrency-cancel-except-newest-with-parent-concurrency-worker",
        workflows=[concurrency_cancel_except_newest_with_parent_concurrency_workflow],
    )
    worker.start()


if __name__ == "__main__":
    main()
