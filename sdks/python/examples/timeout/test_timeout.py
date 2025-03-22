import pytest

from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_execution_timeout(hatchet: Hatchet, worker: Worker) -> None:
    run = timeout_wf.run_no_wait()

    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED_OUT)"):
        await run.aio_result()


@pytest.mark.asyncio(loop_scope="session")
@pytest.mark.parametrize("worker", ["timeout"], indirect=True)
async def test_run_refresh_timeout(hatchet: Hatchet, worker: Worker) -> None:
    result = await refresh_timeout_wf.aio_run()

    assert result["refresh_task"]["status"] == "success"
