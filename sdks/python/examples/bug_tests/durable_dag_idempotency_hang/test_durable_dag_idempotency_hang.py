import pytest
import asyncio
import time

from hatchet_sdk import Hatchet
from examples.bug_tests.durable_dag_idempotency_hang.worker import (
    durable_dag_idempotency_hang_bug_repro_wf,
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
