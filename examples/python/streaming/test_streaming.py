import asyncio
from datetime import datetime, timedelta, timezone
from subprocess import Popen
from typing import Any

import pytest

from examples.streaming.worker import chunks, stream_task
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.listeners.run_event_listener import (
    StepRunEvent,
    StepRunEventType,
)


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
@pytest.mark.parametrize("execution_number", range(1))
@pytest.mark.asyncio(loop_scope="session")
async def test_streaming_ordering_and_completeness(
    execution_number: int,
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    ref = await stream_task.aio_run_no_wait()

    ix = 0
    anna_karenina = ""

    async for chunk in ref.stream():
        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            assert chunks[ix] == chunk.payload
            ix += 1
            anna_karenina += chunk.payload

    assert ix == len(chunks)
    assert anna_karenina == "".join(chunks)

    await ref.aio_result()
