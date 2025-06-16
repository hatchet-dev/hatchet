import pytest

from examples.streaming.worker import stream_task, chunks
from hatchet_sdk.clients.listeners.run_event_listener import (
    StepRunEventType,
)


@pytest.mark.parametrize("execution_number", range(1))
async def test_streaming_ordering_and_completeness(execution_number: int) -> None:
    ref = await stream_task.aio_run_no_wait()

    ix = 0

    async for chunk in ref._wrr.stream():
        if chunk.type != StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            assert ix == len(chunks)
            assert chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_COMPLETED

        assert chunk.payload == chunks[ix], (
            f"Expected chunk {ix} to be '{chunks[ix]}', but got '{chunk}' for execution {execution_number + 1}."
        )

        ix += 1
