import pytest

from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf


@pytest.mark.asyncio()
async def test_run() -> None:
    result = await bulk_parent_wf.aio_run(input=ParentInput(n=12))

    assert len(result["spawn"]["results"]) == 12
