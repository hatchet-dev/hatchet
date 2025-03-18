import pytest

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["fanout"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = parent_wf.run(ParentInput(n=2))
    result = await run.aio_result()
    assert len(result["spawn"]["results"]) == 2
