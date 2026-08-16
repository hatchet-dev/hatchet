import pytest

from hatchet_sdk import Hatchet

from examples.simple.worker import simple


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_pause(hatchet: Hatchet) -> None:
    result = await simple.aio_run()

    assert result["step1"]["step1"] == "step1"
    assert result["step2"]["step2"] == "step2"
    assert result["step3"]["step3"] == "step3"
