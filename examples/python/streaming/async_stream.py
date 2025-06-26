import asyncio

from examples.streaming.worker import stream_task
from hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType


async def main() -> None:
    # > Consume
    ref = await stream_task.aio_run_no_wait()

    async for chunk in ref.stream():
        if chunk.type == StepRunEventType.STEP_RUN_EVENT_TYPE_STREAM:
            print(chunk.payload, flush=True, end="")


if __name__ == "__main__":
    asyncio.run(main())
