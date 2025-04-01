import pytest

from examples.logger.workflow import logging_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio()
async def test_run(hatchet: Hatchet) -> None:
    result = await logging_workflow.aio_run()

    assert result["root_logger"]["status"] == "success"
