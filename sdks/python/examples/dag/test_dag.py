import pytest

from examples.dag.worker import dag_workflow
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["dag"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    result = await dag_workflow.aio_run_and_get_result()

    one = result["step1"]["random_number"]
    two = result["step2"]["random_number"]
    assert result["step3"]["sum"] == one + two
    assert result["step4"]["step4"] == "step4"
