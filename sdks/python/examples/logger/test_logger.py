import pytest

from examples.logger.workflow import logging_workflow
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="session")
@pytest.mark.parametrize("worker", ["logger"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    result = await logging_workflow.aio_run()

    assert result["step1"]["status"] == "success"
