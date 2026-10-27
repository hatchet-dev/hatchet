import asyncio
import json
import uuid

from examples.durable_streams.worker import ProgressInput, durable_streams_progress
from hatchet_sdk import Hatchet

hatchet = Hatchet()


async def consume() -> None:
    # > Consume
    topic = f"progress-{uuid.uuid4()}"
    await durable_streams_progress.aio_run(
        ProgressInput(topic=topic), wait_for_result=False
    )

    async for event in hatchet.streams.aio_events(topic):
        message = json.loads(event.payload)
        print(f"step {message['step']} of {message['total']}")

        if message["step"] == message["total"]:
            break
    # !!


async def publish_from_anywhere() -> None:
    # > Publish from anywhere
    await hatchet.streams.aio_publish(
        "orders", json.dumps({"order_id": "ord_123", "status": "shipped"})
    )
    # !!


def publish_sync() -> None:
    # > Publish synchronously
    hatchet.streams.publish(
        "orders", json.dumps({"order_id": "ord_123", "status": "shipped"})
    )

    for event in hatchet.streams.events("orders"):
        print(event.payload.decode())
        break
    # !!


async def resume_from_cursor(topic: str) -> None:
    # > Resume from a cursor
    cursor = None

    async for event in hatchet.streams.aio_events(topic):
        print(event.payload.decode())
        cursor = event.cursor
        break

    async for event in hatchet.streams.aio_events(topic, cursor=cursor):
        print(event.payload.decode())
    # !!


async def stop_reading(topic: str) -> None:
    # > Stop reading
    async def read() -> None:
        async for event in hatchet.streams.aio_events(topic):
            print(event.payload.decode())

    try:
        await asyncio.wait_for(read(), timeout=10)
    except asyncio.TimeoutError:
        pass
    # !!


async def namespaces() -> None:
    # > Namespaces
    await hatchet.streams.aio_publish("updates", "hello", namespace="customer-a")

    async for event in hatchet.streams.aio_events("updates", namespace="customer-a"):
        print(event.payload.decode())
        break
    # !!


async def reset_topic() -> None:
    # > Reset a topic
    await hatchet.streams.aio_reset("orders")

    async for event in hatchet.streams.aio_events("orders"):
        print(event.payload.decode())
    # !!


async def main() -> None:
    await consume()
    await publish_from_anywhere()
    await stop_reading("orders")
    await namespaces()


if __name__ == "__main__":
    asyncio.run(main())
