import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["dag"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow("DagWorkflow", {})
    result = await run.result()

    one = result["step1"]["rando"]
    two = result["step2"]["rando"]
    assert result["step3"]["sum"] == one + two
    assert result["step4"]["step4"] == "step4"
