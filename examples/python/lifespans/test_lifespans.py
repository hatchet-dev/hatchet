import pytest

from examples.lifespans.simple import Lifespan, lifespan_task

@pytest.mark.asyncio(loop_scope="session")
async def test_lifespans() -> None:
    result = await lifespan_task.aio_run()

    assert isinstance(result, Lifespan)
    assert result.pi == 3.14
    assert result.foo == "bar"
