import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("TimeoutWorkflow", {})
    try:
        await run.result()
        assert False, "Expected workflow to timeout"
    except Exception as e:
        assert str(e) == "Workflow Errors: ['TIMED_OUT']"


@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_refresh_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("RefreshTimeoutWorkflow", {})
    result = await run.result()
    assert result["step1"]["status"] == "success"
