import pytest

from examples.timeout.worker import refresh_timeout_wf, timeout_wf
from hatchet_sdk import Hatchet


@pytest.mark.asyncio()
async def test_execution_timeout(hatchet: Hatchet) -> None:
    run = timeout_wf.run_no_wait()

    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED_OUT)"):
        await run.aio_result()


@pytest.mark.asyncio()
async def test_run_refresh_timeout(hatchet: Hatchet) -> None:
    result = await refresh_timeout_wf.aio_run()

    assert result["refresh_task"]["status"] == "success"
