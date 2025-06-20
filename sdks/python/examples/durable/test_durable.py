import asyncio

import pytest

from examples.durable.worker import EVENT_KEY, SLEEP_TIME, durable_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_durable(hatchet: Hatchet) -> None:
    ref = durable_workflow.run_no_wait()

    await asyncio.sleep(SLEEP_TIME + 10)

    hatchet.event.push(EVENT_KEY, {})

    result = await ref.aio_result()

    workers = await hatchet.workers.aio_list()

    assert workers.rows

    active_workers = [w for w in workers.rows if w.status == "ACTIVE"]

    assert len(active_workers) == 2
    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker")
        for w in active_workers
    )
    assert any(
        w.name == hatchet.config.apply_namespace("e2e-test-worker_durable")
        for w in active_workers
    )
    assert result["durable_task"]["status"] == "success"
