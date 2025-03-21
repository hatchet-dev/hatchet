import pytest

from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = timeout_wf.run_no_wait()

    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED OUT)"):
        await run.aio_result()


@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_refresh_timeout(hatchet: Hatchet, worker: Worker) -> None:
    result = await refresh_timeout_wf.aio_run()

    assert result["refresh_task"]["status"] == "success"
