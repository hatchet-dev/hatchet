import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


# > Queue Oldest
class WorkflowInput(BaseModel):
    group: str


concurrency_queue_oldest_workflow = hatchet.workflow(
    name="ConcurrencyQueueOldest",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.QUEUE_OLDEST,
    ),
    input_validator=WorkflowInput,
)


@concurrency_queue_oldest_workflow.task()
async def step1(input: WorkflowInput, ctx: Context) -> None:
    for _ in range(30):
        await asyncio.sleep(0.10)
