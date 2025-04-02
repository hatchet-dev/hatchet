import asyncio
import time

import pytest

from examples.rate_limit.worker import rate_limit_workflow


@pytest.mark.skip(reason="The timing for this test is not reliable")
@pytest.mark.asyncio()
async def test_run() -> None:

    run1 = rate_limit_workflow.run_no_wait()
    run2 = rate_limit_workflow.run_no_wait()
    run3 = rate_limit_workflow.run_no_wait()

    start_time = time.time()

    await asyncio.gather(run1.aio_result(), run2.aio_result(), run3.aio_result())

    end_time = time.time()

    total_time = end_time - start_time

    assert (
        1 <= total_time <= 5
    ), f"Expected runtime to be a bit more than 1 seconds, but it took {total_time:.2f} seconds"
