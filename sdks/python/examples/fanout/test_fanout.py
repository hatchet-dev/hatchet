import pytest

from examples.fanout.worker import ParentInput, parent_wf


@pytest.mark.asyncio()
async def test_run() -> None:
    result = await parent_wf.aio_run(ParentInput(n=2))

    assert len(result["spawn"]["results"]) == 2
