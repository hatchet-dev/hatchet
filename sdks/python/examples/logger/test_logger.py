import pytest

from examples.logger.workflow import logging_workflow
from hatchet_sdk import Hatchet


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="function")
async def test_run(hatchet: Hatchet) -> None:
    result = await logging_workflow.aio_run()

    assert result["root_logger"]["status"] == "success"
