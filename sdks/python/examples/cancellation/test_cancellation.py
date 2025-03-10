import pytest

from examples.cancellation.worker import wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["cancellation"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = wf.run()

    result = await run.aio_result()
    # TODO is this the expected result for a timed out run...
    assert result == {}
