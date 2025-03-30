import pytest

from examples.fanout.worker import ParentInput, parent_wf
from hatchet_sdk import Hatchet


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="function")
async def test_run(hatchet: Hatchet) -> None:
    result = await parent_wf.aio_run(ParentInput(n=2))

    assert len(result["spawn"]["results"]) == 2
