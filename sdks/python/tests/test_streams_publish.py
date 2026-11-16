import asyncio
import base64
import json
import threading
import types
from contextlib import suppress
from typing import Any

import pytest

from hatchet_sdk.config import ClientConfig
from hatchet_sdk.contracts.v1 import streams_pb2 as streams_protos
from hatchet_sdk.features import streams as streams_module
from hatchet_sdk.features.streams import StreamsClient


def fake_token() -> str:
    claims = {
        "sub": "tenant",
        "server_url": "http://localhost:8080",
        "grpc_broadcast_address": "localhost:7070",
    }
    encoded = base64.urlsafe_b64encode(json.dumps(claims).encode()).decode()
    return f"eyJhbGciOiJub25lIn0.{encoded.rstrip('=')}.sig"


class FakeStub:
    def __init__(self) -> None:
        self.requests: list[streams_protos.PublishStreamMessagesRequest] = []
        self.in_flight = asyncio.Event()
        self.release = asyncio.Event()
        self.release.set()

    async def PublishBatch(  # noqa: N802
        self, request: streams_protos.PublishStreamMessagesRequest, metadata: Any
    ) -> None:
        self.requests.append(request)
        self.in_flight.set()
        await self.release.wait()

    def payloads(self) -> list[list[bytes]]:
        return [[e.payload for e in r.entries] for r in self.requests]


class FakeChannel:
    def __init__(self) -> None:
        self.closed = False

    async def close(self) -> None:
        self.closed = True


@pytest.fixture
def channels(monkeypatch: pytest.MonkeyPatch) -> list[FakeChannel]:
    opened: list[FakeChannel] = []

    def new_conn(*_: Any) -> FakeChannel:
        opened.append(FakeChannel())
        return opened[-1]

    monkeypatch.setattr(streams_module, "new_conn", new_conn)
    return opened


@pytest.fixture
def stub(monkeypatch: pytest.MonkeyPatch, channels: list[FakeChannel]) -> FakeStub:
    fake = FakeStub()
    monkeypatch.setattr(streams_module, "V1StreamsStub", lambda _: fake)
    return fake


@pytest.fixture
def client() -> StreamsClient:
    return StreamsClient(ClientConfig(token=fake_token()))


async def test_publishes_queued_behind_a_request_share_the_next(
    client: StreamsClient, stub: FakeStub
) -> None:
    stub.release.clear()
    first = asyncio.create_task(client.publish("topic", "a"))
    await stub.in_flight.wait()

    rest = [asyncio.create_task(client.publish("topic", m)) for m in ["b", "c"]]
    await asyncio.sleep(0)
    stub.release.set()
    await asyncio.gather(first, *rest)

    assert stub.payloads() == [[b"a"], [b"b", b"c"]]
    assert [r.first_producer_seq for r in stub.requests] == [0, 1]
    assert stub.requests[0].producer_id == stub.requests[1].producer_id


async def test_cancelling_an_in_flight_publish_leaves_the_topic_usable(
    client: StreamsClient, stub: FakeStub
) -> None:
    stub.release.clear()
    first = asyncio.create_task(client.publish("topic", "a"))
    await stub.in_flight.wait()

    first.cancel()
    with suppress(asyncio.CancelledError):
        await first

    stub.release.set()
    await asyncio.wait_for(client.publish("topic", "b"), timeout=1)

    assert stub.payloads() == [[b"a"], [b"b"]]
    assert [r.first_producer_seq for r in stub.requests] == [0, 1]


async def test_cancelling_a_queued_publish_drops_it(
    client: StreamsClient, stub: FakeStub
) -> None:
    stub.release.clear()
    first = asyncio.create_task(client.publish("topic", "a"))
    await stub.in_flight.wait()

    queued = asyncio.create_task(client.publish("topic", "b"))
    await asyncio.sleep(0)
    queued.cancel()
    with suppress(asyncio.CancelledError):
        await queued

    stub.release.set()
    await first
    await asyncio.wait_for(client.publish("topic", "c"), timeout=1)

    assert stub.payloads() == [[b"a"], [b"c"]]
    assert [r.first_producer_seq for r in stub.requests] == [0, 1]


async def test_large_payloads_upload_on_the_upload_pool(
    client: StreamsClient, stub: FakeStub, monkeypatch: pytest.MonkeyPatch
) -> None:
    threads: list[str] = []

    def upload(payload: bytes) -> str:
        threads.append(threading.current_thread().name)
        return "ref"

    monkeypatch.setattr(client, "_upload_payload", upload)

    await client.publish("topic", b"x" * (streams_module.MAX_INLINE_PAYLOAD_BYTES + 1))

    assert stub.requests[0].entries[0].payload_ref == "ref"
    assert threads[0].startswith("hatchet-stream-upload")


def test_loop_shutdown_closes_its_channel(
    client: StreamsClient, stub: FakeStub, channels: list[FakeChannel]
) -> None:
    asyncio.run(client.publish("topic", "a"))

    assert [c.closed for c in channels] == [True]
    assert client._publishers == {}


def test_a_loop_closed_without_shutdown_is_dropped_on_the_next_publish(
    client: StreamsClient, stub: FakeStub, channels: list[FakeChannel]
) -> None:
    loop = asyncio.new_event_loop()
    loop.run_until_complete(client.publish("topic", "a"))
    hook = client._publishers[loop].shutdown_hook
    loop.close()

    asyncio.run(client.publish("topic", "b"))

    assert isinstance(hook, types.AsyncGeneratorType) and hook.ag_frame is None
    assert client._publishers == {}
    assert [c.closed for c in channels] == [False, True]
    assert stub.payloads() == [[b"a"], [b"b"]]
