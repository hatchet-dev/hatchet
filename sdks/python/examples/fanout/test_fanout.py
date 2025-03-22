import pytest

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="session")
@pytest.mark.parametrize("worker", ["fanout"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    result = await parent_wf.aio_run(ParentInput(n=2))

    assert len(result["spawn"]["results"]) == 2
