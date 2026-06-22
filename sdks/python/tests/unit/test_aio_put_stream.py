from __future__ import annotations

import asyncio
from collections.abc import Awaitable
from typing import cast

import grpc
import grpc.aio
import pytest
import tenacity

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig, TenacityConfig
from hatchet_sdk.context.context import Context
from hatchet_sdk.contracts.events_pb2 import (
    PutStreamEventRequest,
    PutStreamEventResponse,
)
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub


def _make_grpc_error(code: grpc.StatusCode, details: str = "") -> grpc.aio.AioRpcError:
    empty: grpc.aio.Metadata = grpc.aio.Metadata()
    return grpc.aio.AioRpcError(code, empty, empty, details)


class _GeneratedAioUnaryCall:
    """Matches grpc.aio generated unary calls: sync call, awaitable result."""

    def __init__(self, failures_before_success: int) -> None:
        self.failures_before_success = failures_before_success
        self.calls = 0
        self.requests: list[PutStreamEventRequest] = []
        self.metadata: list[tuple[tuple[str, str]]] = []

    def __call__(
        self,
        request: PutStreamEventRequest,
        *,
        metadata: tuple[tuple[str, str]],
    ) -> Awaitable[PutStreamEventResponse]:
        self.calls += 1
        self.requests.append(request)
        self.metadata.append(metadata)

        async def response() -> PutStreamEventResponse:
            if self.calls <= self.failures_before_success:
                raise _make_grpc_error(grpc.StatusCode.UNAVAILABLE, "transient")

            return PutStreamEventResponse()

        return response()


class _FakeAioEventsServiceStub:
    def __init__(self, put_stream_event: _GeneratedAioUnaryCall) -> None:
        self.PutStreamEvent = put_stream_event


def _event_client(aio_stub: _FakeAioEventsServiceStub) -> EventClient:
    client = EventClient.__new__(EventClient)
    client.client_config = ClientConfig.model_construct(
        tenant_id="tenant",
        token="token",
        namespace="",
        server_url="http://localhost",
        host_port="localhost:7070",
        tenacity=TenacityConfig(max_attempts=3, wait=tenacity.wait_none),
    )
    client.token = "token"
    client.namespace = ""
    client._aio_client = cast(EventsServiceStub, aio_stub)
    client._retrying_aio_put_stream_event = tenacity_retry(
        client._put_stream_event, client.client_config.tenacity
    )

    return client


@pytest.mark.parametrize(
    ("data", "expected_message"),
    [
        ("hello", b"hello"),
        (b"hello", b"hello"),
    ],
)
async def test_aio_stream_retries_generated_aio_callable(
    data: str | bytes, expected_message: bytes
) -> None:
    put_stream_event = _GeneratedAioUnaryCall(failures_before_success=2)
    client = _event_client(_FakeAioEventsServiceStub(put_stream_event))

    await client.aio_stream(data, step_run_id="step-run-id", index=7)

    assert put_stream_event.calls == 3
    assert [request.task_run_external_id for request in put_stream_event.requests] == [
        "step-run-id",
        "step-run-id",
        "step-run-id",
    ]
    assert [request.message for request in put_stream_event.requests] == [
        expected_message,
        expected_message,
        expected_message,
    ]
    assert [request.event_index for request in put_stream_event.requests] == [7, 7, 7]
    assert put_stream_event.metadata == [
        (("authorization", "bearer token"),),
        (("authorization", "bearer token"),),
        (("authorization", "bearer token"),),
    ]


class _RecordingEventClient:
    def __init__(self) -> None:
        self.calls: list[tuple[str | bytes, str, int]] = []
        self.both_calls_started = asyncio.Event()
        self.release_sends = asyncio.Event()

    async def aio_stream(self, data: str | bytes, step_run_id: str, index: int) -> None:
        self.calls.append((data, step_run_id, index))
        if len(self.calls) == 2:
            self.both_calls_started.set()
        await self.release_sends.wait()


def _context(event_client: _RecordingEventClient) -> Context:
    ctx = Context.__new__(Context)
    ctx._stream_index = 0
    ctx._step_run_id = "step-run-id"
    ctx._event_client = cast(EventClient, event_client)

    return ctx


async def test_aio_put_stream_assigns_index_before_async_send() -> None:
    event_client = _RecordingEventClient()
    ctx = _context(event_client)

    tasks = [
        asyncio.create_task(ctx.aio_put_stream("first")),
        asyncio.create_task(ctx.aio_put_stream(b"second")),
    ]

    try:
        await asyncio.wait_for(event_client.both_calls_started.wait(), timeout=1)

        assert event_client.calls == [
            ("first", "step-run-id", 0),
            (b"second", "step-run-id", 1),
        ]
    finally:
        event_client.release_sends.set()
        await asyncio.gather(*tasks)
