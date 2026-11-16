import pytest
import asyncio
import time

from hatchet_sdk import Hatchet
from examples.bug_tests.durable_evict_timeout.worker import evictable_durable


@pytest.mark.asyncio(loop_scope="session")
async def test_eviction_execution_timeout(hatchet: Hatchet) -> None:
    start = time.time()

    with pytest.raises(Exception) as exc_info:
        await evictable_durable.aio_run()

    assert (
        time.time() - start < 20
    ), "Eviction did not occur within the expected time frame"
