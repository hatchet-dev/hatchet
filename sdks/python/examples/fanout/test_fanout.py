import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["fanout"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("Parent", {"n": 2})
    result = await run.result()
    assert len(result["spawn"]["results"]) == 2


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["fanout"], indirect=True)
async def test_run2(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("Parent", {"n": 2})
    result = await run.result()
    assert len(result["spawn"]["results"]) == 2
