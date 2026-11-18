import time

import pytest

from examples.bug_tests.durable_evict_timeout.worker import evictable_durable
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_eviction_execution_timeout(hatchet: Hatchet) -> None:
    start = time.time()

    with pytest.raises(FailedTaskRunExceptionGroup):
        await evictable_durable.aio_run()

    assert (
        time.time() - start < 20
    ), "Eviction did not occur within the expected time frame"
