import pytest

from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["bulk_fanout"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = bulk_parent_wf.run(input=ParentInput(n=12))
    result = await run.aio_result()

    assert len(result["spawn"]["results"]) == 12
