import pytest
import asyncio
import time

from hatchet_sdk import Hatchet
from examples.bug_tests.durable_dag_idempotency_hang.worker import (
    durable_dag_idempotency_hang_bug_repro_wf,
    durable_dag_status_based_idempotency_wf,
    IdempotencyHangInput,
)
from uuid import uuid4


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_idempotency_hang(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    async with asyncio.timeout(10):
        result = await durable_dag_idempotency_hang_bug_repro_wf.aio_run(
            wait_for_result=True,
            input=IdempotencyHangInput(key=test_run_id),
            additional_metadata={"test_run_id": test_run_id},
        )

    assert set(result.keys()) == {
        t.name for t in durable_dag_idempotency_hang_bug_repro_wf.tasks
    }


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_dag_status_based_idempotency_key_is_released_on_completion(
    hatchet: Hatchet,
) -> None:
    workflow_input = IdempotencyHangInput(key=str(uuid4()))

    async with asyncio.timeout(20):
        first_result = await durable_dag_status_based_idempotency_wf.aio_run(
            workflow_input
        )
        second_result = await durable_dag_status_based_idempotency_wf.aio_run(
            workflow_input
        )

    assert first_result == second_result
