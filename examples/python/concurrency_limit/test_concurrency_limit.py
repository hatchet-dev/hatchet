import pytest

from examples.concurrency_limit.worker import WorkflowInput, concurrency_limit_workflow
from hatchet_sdk.workflow_run import WorkflowRunRef

@pytest.mark.asyncio(loop_scope="session")
@pytest.mark.skip(reason="The timing for this test is not reliable")
async def test_run() -> None:
    num_runs = 6
    runs: list[WorkflowRunRef] = []

    # Start all runs
    for i in range(1, num_runs + 1):
        run = concurrency_limit_workflow.run_no_wait(
            WorkflowInput(run=i, group_key=str(i))
        )
        runs.append(run)

    # Wait for all results
    successful_runs = []
    cancelled_runs = []

    # Process each run individually
    for i, run in enumerate(runs, start=1):
        try:
            result = await run.aio_result()
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
