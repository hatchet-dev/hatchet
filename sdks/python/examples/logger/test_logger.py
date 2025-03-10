import pytest

from examples.logger.workflow import wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["logger"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = wf.run()

    result = await run.aio_result()
    assert result["step1"]["status"] == "success"
