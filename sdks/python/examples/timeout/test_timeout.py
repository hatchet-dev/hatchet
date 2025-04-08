import pytest

from examples.timeout.worker import refresh_timeout_wf, timeout_wf


@pytest.mark.asyncio(loop_scope="session")
async def test_execution_timeout() -> None:
    run = timeout_wf.run_no_wait()

    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED_OUT)"):
        await run.aio_result()


@pytest.mark.asyncio(loop_scope="session")
async def test_run_refresh_timeout() -> None:
    result = await refresh_timeout_wf.aio_run()

    assert result["refresh_task"]["status"] == "success"
