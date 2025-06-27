from subprocess import Popen
from typing import Any

import pytest

from examples.streaming.worker import chunks, stream_task
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType


@pytest.mark.parametrize(
    "on_demand_worker",
    [
        (
            ["poetry", "run", "python", "examples/streaming/worker.py", "--slots", "1"],
            8008,
        )
    ],
    indirect=True,
)
@pytest.mark.parametrize("execution_number", range(5))  # run test multiple times
@pytest.mark.asyncio(loop_scope="session")
async def test_streaming_ordering_and_completeness(
    execution_number: int,
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    ref = await stream_task.aio_run_no_wait()

    ix = 0
    anna_karenina = ""

    async for chunk in hatchet.subscribe_to_stream(ref.workflow_run_id):
        assert chunks[ix] == chunk
        ix += 1
        anna_karenina += chunk

    assert ix == len(chunks)
    assert anna_karenina == "".join(chunks)

    await ref.aio_result()
