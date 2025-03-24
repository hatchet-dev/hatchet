import pytest

from examples.cancellation.worker import wf
from hatchet_sdk import Hatchet


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(loop_scope="session")
async def test_run(hatchet: Hatchet) -> None:
    with pytest.raises(Exception, match="(Task exceeded timeout|TIMED_OUT)"):
        await wf.aio_run()
