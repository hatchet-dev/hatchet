from __future__ import annotations

from collections.abc import Awaitable
from typing import cast

import grpc
import grpc.aio
import pytest
import tenacity

from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import ClientConfig, TenacityConfig
from hatchet_sdk.contracts.events_pb2 import (
    BulkPushEventRequest,
    Event,
    Events,
    PushEventRequest,
)
from hatchet_sdk.contracts.events_pb2_grpc import EventsServiceStub
from hatchet_sdk.types.trigger import BulkPushEventWithMetadata


def _make_grpc_error(code: grpc.StatusCode, details: str = "") -> grpc.aio.AioRpcError:
    empty: grpc.aio.Metadata = grpc.aio.Metadata()
    return grpc.aio.AioRpcError(code, empty, empty, details)


class _GeneratedAioUnaryCall:
    """Matches grpc.aio generated unary calls: sync call, awaitable result."""

    def __init__(
        self,
        failures_before_success: int | None,
        failure_code: grpc.StatusCode = grpc.StatusCode.UNAVAILABLE,
    ) -> None:
        self.failures_before_success = failures_before_success
        self.failure_code = failure_code
        self.calls = 0
        self.requests: list[PushEventRequest | BulkPushEventRequest] = []

    def __call__(
        self,
        request: PushEventRequest | BulkPushEventRequest,
        *,
        metadata: tuple[tuple[str, str]],
    ) -> Awaitable[Event | Events]:
        self.calls += 1
        self.requests.append(request)

        async def response() -> Event | Events:
            if (
                self.failures_before_success is None
                or self.calls <= self.failures_before_success
            ):
                raise _make_grpc_error(self.failure_code, "grpc failure")

            if isinstance(request, BulkPushEventRequest):
                return Events(events=[Event(key=e.key) for e in request.events])

            return Event(key=request.key)

        return response()


class _FakeAioEventsServiceStub:
    def __init__(
        self,
        push: _GeneratedAioUnaryCall | None = None,
        bulk_push: _GeneratedAioUnaryCall | None = None,
    ) -> None:
        self.Push = push or _GeneratedAioUnaryCall(failures_before_success=0)
        self.BulkPush = bulk_push or _GeneratedAioUnaryCall(failures_before_success=0)


def _event_client(
    aio_stub: _FakeAioEventsServiceStub, max_attempts: int = 3
) -> EventClient:
    client = EventClient.__new__(EventClient)
    client.client_config = ClientConfig.model_construct(
        tenant_id="tenant",
        token="token",
        namespace="",
        server_url="http://localhost",
        host_port="localhost:7070",
        tenacity=TenacityConfig(max_attempts=max_attempts, wait=tenacity.wait_none),
    )
    client.token = "token"
    client.namespace = ""
    client._aio_client = cast(EventsServiceStub, aio_stub)
    client._retrying_aio_push_event = tenacity_retry(
        client._push_event, client.client_config.tenacity
    )
    client._retrying_aio_bulk_push_event = tenacity_retry(
        client._bulk_push_event, client.client_config.tenacity
    )

    return client


async def test_aio_push_retries_generated_aio_callable_on_retryable_error() -> None:
    push = _GeneratedAioUnaryCall(failures_before_success=2)
    client = _event_client(_FakeAioEventsServiceStub(push=push))

    event = await client.aio_push("event:key", {"foo": "bar"})

    assert push.calls == 3
    assert event.key == "event:key"


async def test_aio_push_does_not_retry_non_retryable_error() -> None:
    push = _GeneratedAioUnaryCall(
        failures_before_success=None, failure_code=grpc.StatusCode.INVALID_ARGUMENT
    )
    client = _event_client(_FakeAioEventsServiceStub(push=push))

    with pytest.raises(grpc.aio.AioRpcError) as exc_info:
        await client.aio_push("event:key", {"foo": "bar"})

    assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT
    assert push.calls == 1


async def test_aio_push_propagates_error_after_exhausting_attempts() -> None:
    push = _GeneratedAioUnaryCall(failures_before_success=None)
    client = _event_client(_FakeAioEventsServiceStub(push=push), max_attempts=3)

    with pytest.raises(grpc.aio.AioRpcError) as exc_info:
        await client.aio_push("event:key", {"foo": "bar"})

    assert exc_info.value.code() == grpc.StatusCode.UNAVAILABLE
    assert push.calls == 3


async def test_aio_bulk_push_retries_generated_aio_callable_on_retryable_error() -> (
    None
):
    bulk_push = _GeneratedAioUnaryCall(failures_before_success=2)
    client = _event_client(_FakeAioEventsServiceStub(bulk_push=bulk_push))

    events = await client.aio_bulk_push(
        [
            BulkPushEventWithMetadata(key="event:1"),
            BulkPushEventWithMetadata(key="event:2"),
        ]
    )

    assert bulk_push.calls == 3
    assert [e.key for e in events] == ["event:1", "event:2"]


async def test_aio_bulk_push_does_not_retry_non_retryable_error() -> None:
    bulk_push = _GeneratedAioUnaryCall(
        failures_before_success=None, failure_code=grpc.StatusCode.INVALID_ARGUMENT
    )
    client = _event_client(_FakeAioEventsServiceStub(bulk_push=bulk_push))

    with pytest.raises(grpc.aio.AioRpcError) as exc_info:
        await client.aio_bulk_push([BulkPushEventWithMetadata(key="event:1")])

    assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT
    assert bulk_push.calls == 1


async def test_aio_bulk_push_propagates_error_after_exhausting_attempts() -> None:
    bulk_push = _GeneratedAioUnaryCall(failures_before_success=None)
    client = _event_client(
        _FakeAioEventsServiceStub(bulk_push=bulk_push), max_attempts=3
    )

    with pytest.raises(grpc.aio.AioRpcError) as exc_info:
        await client.aio_bulk_push([BulkPushEventWithMetadata(key="event:1")])

    assert exc_info.value.code() == grpc.StatusCode.UNAVAILABLE
    assert bulk_push.calls == 3
