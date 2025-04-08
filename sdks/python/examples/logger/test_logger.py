import pytest

from examples.logger.workflow import logging_workflow


@pytest.mark.asyncio(loop_scope="session")
async def test_run() -> None:
    result = await logging_workflow.aio_run()

    assert result["root_logger"]["status"] == "success"
