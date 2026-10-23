import asyncio
import uuid

import pytest

from examples.bug_tests.single_task_workflow_concurrency_retry_backoff.worker import (
    RETRY_BACKOFF_SECONDS,
    RetryBackoffInput,
    single_task_workflow_concurrency_retry_backoff,
)


@pytest.mark.timeout(60, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_concurrency_holds_slot_through_retry_backoff() -> None:
    concurrency_key = str(uuid.uuid4())

    run_a = await single_task_workflow_concurrency_retry_backoff.aio_run(
        RetryBackoffInput(concurrency_key=concurrency_key, fail_first_attempt=True),
        wait_for_result=False,
    )

    await asyncio.sleep(RETRY_BACKOFF_SECONDS / 2)

    run_b = await single_task_workflow_concurrency_retry_backoff.aio_run(
        RetryBackoffInput(concurrency_key=concurrency_key, fail_first_attempt=False),
        wait_for_result=False,
    )

    result_a, result_b = await asyncio.gather(run_a.aio_result(), run_b.aio_result())

    attempt_a = result_a["record_attempt"]
    attempt_b = result_b["record_attempt"]

    assert attempt_a["retry_count"] == 1
    assert attempt_b["retry_count"] == 0
    assert attempt_a["started_at"] < attempt_b["started_at"], (
        "run B started while run A was waiting out its retry backoff, "
        "so the workflow-level concurrency slot was released between retries"
    )
