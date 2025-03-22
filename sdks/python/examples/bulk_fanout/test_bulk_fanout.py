import pytest

from examples.bulk_fanout.worker import ParentInput, bulk_parent_wf
from hatchet_sdk import Hatchet


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    result = await bulk_parent_wf.aio_run(input=ParentInput(n=12))

    assert len(result["spawn"]["results"]) == 12
