import time

import pytest

from hatchet_sdk import Hatchet, Worker
from hatchet_sdk.workflow_run import WorkflowRunRef


# requires scope module or higher for shared event loop
@pytest.mark.skip(reason="The timing for this test is not reliable")
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["concurrency_limit_rr"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    num_groups = 2
    runs: list[WorkflowRunRef] = []

    # Start all runs
    for i in range(1, num_groups + 1):
        run = hatchet.admin.run_workflow("ConcurrencyDemoWorkflowRR", {"group": i})
        runs.append(run)
        run = hatchet.admin.run_workflow("ConcurrencyDemoWorkflowRR", {"group": i})
        runs.append(run)

    # Wait for all results
    successful_runs = []
    cancelled_runs = []

    start_time = time.time()

    # Process each run individually
    for i, run in enumerate(runs, start=1):
        try:
            result = await run.result()
            successful_runs.append((i, result))
        except Exception as e:
            if "CANCELLED_BY_CONCURRENCY_LIMIT" in str(e):
                cancelled_runs.append((i, str(e)))
            else:
                raise  # Re-raise if it's an unexpected error

    end_time = time.time()
    total_time = end_time - start_time

    # Check that we have the correct number of successful and cancelled runs
    assert (
        len(successful_runs) == 4
    ), f"Expected 4 successful runs, got {len(successful_runs)}"
    assert (
        len(cancelled_runs) == 0
    ), f"Expected 0 cancelled run, got {len(cancelled_runs)}"

    # Check that the total time is close to 2 seconds
    assert (
        3.8 <= total_time <= 7
    ), f"Expected runtime to be about 4 seconds, but it took {total_time:.2f} seconds"

    print(f"Total execution time: {total_time:.2f} seconds")
