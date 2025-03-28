import asyncio

import pytest

from examples.durable.worker import EVENT_KEY, SLEEP_TIME, durable_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_durable(hatchet: Hatchet) -> None:
    ref = durable_workflow.run_no_wait()

    await asyncio.sleep(SLEEP_TIME + 4)

    hatchet.event.push(EVENT_KEY, {})

    result = await ref.aio_result()

    assert result["durable_task"]["status"] == "success"
