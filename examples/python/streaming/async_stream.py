import asyncio

from examples.streaming.worker import hatchet, stream_task
from hatchet_sdk.clients.listeners.run_event_listener import StepRunEventType


async def main() -> None:
    # > Consume
    ref = await stream_task.aio_run_no_wait()

    async for chunk in hatchet.subscribe_to_stream(ref.workflow_run_id):
        print(chunk, flush=True, end="")


if __name__ == "__main__":
    asyncio.run(main())
