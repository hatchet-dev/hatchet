import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["logger"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("LoggingWorkflow", {})
    result = await run.result()
    assert result["step1"]["status"] == "success"
