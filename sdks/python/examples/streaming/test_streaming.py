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
    streamed_chunks: list[tuple[int, StepRunEvent]] = []

    async for chunk in ref._wrr.stream():
        print(
            f"Received chunk {ix}: {chunk} at {datetime.now(timezone(timedelta(hours=-4), name='EST'))}"
        )
        streamed_chunks.append((ix, chunk))
        ix += 1

    await asyncio.sleep(10)

    assert ix == len(chunks) + 1

    for ix, chunk in streamed_chunks:
        if chunk.type != StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            assert ix == len(chunks)
            assert chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED
        else:
            assert (
                chunk.payload == chunks[ix]
            ), f"Expected chunk {ix} to be '{chunks[ix]}', but got '{chunk}' for execution {execution_number + 1}."

    await ref.aio_result()
