import pytest

from examples.dag.worker import dag_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    result = await dag_workflow.aio_run()

    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]
    assert result["step3"]["sum"] == one + two
    assert result["step4"]["step4"] == "step4"
