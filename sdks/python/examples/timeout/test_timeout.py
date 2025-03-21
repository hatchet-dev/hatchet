import pytest

from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = timeout_wf.run()

    try:
        await run.aio_result()
        assert False, "Expected workflow to timeout"
    except Exception as e:
        assert str(e) == "Workflow Errors: ['TIMED_OUT']"


@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_refresh_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = refresh_timeout_wf.run()

    result = await run.aio_result()
    assert result["refresh_task"]["status"] == "success"
