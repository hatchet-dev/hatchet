import asyncio
import json
import uuid

import grpc
import pytest

from examples.durable_streams.worker import (
    ProgressInput,
    durable_streams_retrying_progress,
)
from hatchet_sdk import Hatchet, StreamEvent


def scope(name: str) -> str:
    return f"{name}_{uuid.uuid4()}"


async def collect(
    hatchet: Hatchet, topic: str, count: int, **kwargs: str
) -> list[StreamEvent]:
    events: list[StreamEvent] = []

    async def read() -> None:
        async for event in hatchet.streams.subscribe(topic, **kwargs):
            events.append(event)
            if len(events) >= count:
                break

    await asyncio.wait_for(read(), timeout=30)

    return events


def payloads(events: list[StreamEvent]) -> list[str]:
    return [e.payload.decode() for e in events]


@pytest.mark.asyncio(loop_scope="session")
async def test_reads_from_the_beginning_then_live(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_from_beginning")

    await hatchet.streams.publish(topic, "published-before-consume")

    async def publish_later() -> None:
        await asyncio.sleep(0.2)
        for message in ["live-1", "live-2"]:
            await hatchet.streams.publish(topic, message)

    publishing = asyncio.create_task(publish_later())
    events = await collect(hatchet, topic, 3)
    await publishing

    assert payloads(events) == ["published-before-consume", "live-1", "live-2"]


@pytest.mark.asyncio(loop_scope="session")
async def test_resumes_from_a_cursor(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_resume_cursor")

    for message in ["one", "two", "three", "four"]:
        await hatchet.streams.publish(topic, message)

    first_two = await collect(hatchet, topic, 2)
    assert payloads(first_two) == ["one", "two"]

    rest = await collect(hatchet, topic, 2, cursor=first_two[-1].cursor)
    assert payloads(rest) == ["three", "four"]


@pytest.mark.asyncio(loop_scope="session")
async def test_reset_starts_new_readers_after_it(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_reset")

    for message in ["one", "two", "three"]:
        await hatchet.streams.publish(topic, message)

    [first] = await collect(hatchet, topic, 1)
    await hatchet.streams.aio_reset(topic, cursor=first.cursor)
    assert payloads(await collect(hatchet, topic, 2)) == ["two", "three"]

    await hatchet.streams.aio_reset(topic)
    await hatchet.streams.publish(topic, "after-reset")
    assert payloads(await collect(hatchet, topic, 1)) == ["after-reset"]


@pytest.mark.asyncio(loop_scope="session")
async def test_isolates_namespaces(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_namespace")

    await hatchet.streams.publish(topic, "from-a", namespace="ns-a")
    await hatchet.streams.publish(topic, "from-b", namespace="ns-b")

    events = await collect(hatchet, topic, 1, namespace="ns-b")
    assert payloads(events) == ["from-b"]


@pytest.mark.asyncio(loop_scope="session")
async def test_topic_metadata(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_metadata")

    for message in ["first", "second", "third"]:
        await hatchet.streams.publish(topic, message)

    metadata = await hatchet.streams.aio_topic_metadata(topic)

    assert metadata.namespace == ""
    assert metadata.topic == topic
    assert metadata.message_count == 3
    assert metadata.tenant_id == hatchet.tenant_id
    assert metadata.last_published_at is not None

    events = await collect(hatchet, topic, 3)
    assert metadata.latest_cursor == events[2].cursor

    with pytest.raises(grpc.RpcError) as exc:
        await hatchet.streams.aio_topic_metadata(scope("durable_streams_missing"))
    assert exc.value.code() == grpc.StatusCode.NOT_FOUND


@pytest.mark.asyncio(loop_scope="session")
async def test_large_payload(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_large_payload")
    large = b"x" * (5 * 1024 * 1024)

    await hatchet.streams.publish(topic, large)
    await hatchet.streams.publish(topic, "small")

    events = await collect(hatchet, topic, 2)
    assert events[0].payload == large
    assert events[1].payload == b"small"


@pytest.mark.asyncio(loop_scope="session")
async def test_new_readers_skip_a_failed_attempt(hatchet: Hatchet) -> None:
    topic = scope("durable_streams_retry")

    await durable_streams_retrying_progress.aio_run(ProgressInput(topic=topic))

    received = []
    async for event in hatchet.streams.subscribe(topic):
        message = json.loads(event.payload)
        if message.get("done"):
            break
        received.append(message)

    assert received == [{"attempt": 1, "chunk": chunk} for chunk in range(3)]
