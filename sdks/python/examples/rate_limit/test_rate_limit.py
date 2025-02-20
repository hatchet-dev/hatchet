import asyncio
import time

import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.skip(reason="The timing for this test is not reliable")
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["rate_limit"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:

    run1 = hatchet.admin.run_workflow("RateLimitWorkflow", {})
    run2 = hatchet.admin.run_workflow("RateLimitWorkflow", {})
    run3 = hatchet.admin.run_workflow("RateLimitWorkflow", {})

    start_time = time.time()

    await asyncio.gather(run1.result(), run2.result(), run3.result())

    end_time = time.time()

    total_time = end_time - start_time

    assert (
        1 <= total_time <= 5
    ), f"Expected runtime to be a bit more than 1 seconds, but it took {total_time:.2f} seconds"
