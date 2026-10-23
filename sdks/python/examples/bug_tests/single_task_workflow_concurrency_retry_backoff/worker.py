import time
from typing import Any

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()

RETRY_BACKOFF_SECONDS = 4


class RetryBackoffInput(BaseModel):
    concurrency_key: str
    fail_first_attempt: bool


single_task_workflow_concurrency_retry_backoff = hatchet.workflow(
    name="single-task-workflow-concurrency-retry-backoff",
    input_validator=RetryBackoffInput,
    concurrency=ConcurrencyExpression(
        expression="input.concurrency_key",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
)


@single_task_workflow_concurrency_retry_backoff.task(
    retries=1,
    backoff_factor=float(RETRY_BACKOFF_SECONDS),
    backoff_max_seconds=RETRY_BACKOFF_SECONDS,
)
async def record_attempt(input: RetryBackoffInput, ctx: Context) -> dict[str, Any]:
    started_at = time.time()

    if input.fail_first_attempt and ctx.retry_count == 0:
        raise ValueError(
            "failing the first attempt so the run waits out a retry backoff"
        )

    return {"started_at": started_at, "retry_count": ctx.retry_count}
