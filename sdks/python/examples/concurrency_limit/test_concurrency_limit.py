import pytest

from hatchet_sdk import Hatchet, Worker
from hatchet_sdk.workflow_run import WorkflowRunRef


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.skip(reason="The timing for this test is not reliable")
@pytest.mark.parametrize("worker", ["concurrency_limit"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    num_runs = 6
    runs: list[WorkflowRunRef] = []

    # Start all runs
    for i in range(1, num_runs + 1):
        run = hatchet.admin.run_workflow("ConcurrencyDemoWorkflow", {"run": i})
        runs.append(run)

    # Wait for all results
    successful_runs = []
    cancelled_runs = []

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

    # Check that we have the correct number of successful and cancelled runs
    assert (
        len(successful_runs) == 5
    ), f"Expected 5 successful runs, got {len(successful_runs)}"
    assert (
        len(cancelled_runs) == 1
    ), f"Expected 1 cancelled run, got {len(cancelled_runs)}"
