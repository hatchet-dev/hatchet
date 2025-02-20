import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["async"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("AsyncWorkflow", {})
    result = await run.result()
    assert result["step1"]["test"] == "test"


@pytest.mark.parametrize("worker", ["async"], indirect=True)
@pytest.mark.skip(reason="Skipping this test until we can dedicate more time to debug")
@pytest.mark.asyncio(scope="session")
async def test_run_async(aiohatchet: Hatchet, worker: Worker) -> None:
    run = await aiohatchet.admin.aio.run_workflow("AsyncWorkflow", {})
    result = await run.result()
    assert result["step1"]["test"] == "test"
