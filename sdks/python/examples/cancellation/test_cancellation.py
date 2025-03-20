import pytest

from examples.cancellation.worker import wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["cancellation"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED OUT)"):
        await wf.aio_run_and_get_result()
