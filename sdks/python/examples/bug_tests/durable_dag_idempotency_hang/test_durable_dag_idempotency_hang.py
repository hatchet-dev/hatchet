import pytest
import asyncio
import time

from hatchet_sdk import Hatchet
from examples.bug_tests.durable_dag_idempotency_hang.worker import (
    durable_dag_idempotency_hang_bug_repro_wf,
)


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_idempotency_hang(hatchet: Hatchet) -> None:
    async with asyncio.timeout(10):
        result = await durable_dag_idempotency_hang_bug_repro_wf.aio_run(
            wait_for_result=True
        )

    print(result)

    assert False
