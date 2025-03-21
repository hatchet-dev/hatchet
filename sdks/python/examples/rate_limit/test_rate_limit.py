import asyncio
import time

import pytest

from examples.rate_limit.worker import rate_limit_workflow
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.skip(reason="The timing for this test is not reliable")
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["rate_limit"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:

    run1 = rate_limit_workflow.run()
    run2 = rate_limit_workflow.run()
    run3 = rate_limit_workflow.run()

    start_time = time.time()

    await asyncio.gather(run1.aio_result(), run2.aio_result(), run3.aio_result())

    end_time = time.time()

    total_time = end_time - start_time

    assert (
        1 <= total_time <= 5
    ), f"Expected runtime to be a bit more than 1 seconds, but it took {total_time:.2f} seconds"
