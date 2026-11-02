import pytest

from examples.bug_tests.durable_stream_reconnect_mid_wait.worker import (
    durable_stream_reconnect_mid_wait,
)
from hatchet_sdk import Hatchet


@pytest.mark.timeout(180, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_durable_wait_resumes_after_stream_reconnect(hatchet: Hatchet) -> None:
    result = await durable_stream_reconnect_mid_wait.aio_run()

    assert (
        result["invocation_count"] == 1
    ), "the run was evicted and replayed instead of resuming on the reconnected stream"
    assert result["second_sleep_seconds"] < 30, (
        "the second durable sleep stalled after the reconnect: "
        f"it took {result['second_sleep_seconds']:.1f}s"
    )
