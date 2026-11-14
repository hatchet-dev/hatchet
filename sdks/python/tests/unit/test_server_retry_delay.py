"""Retries wait the delay the server asks for, through a real in-process gRPC server.

The server fails calls with a google.rpc.Status, so a RetryInfo detail reaches the client in the
`grpc-status-details-bin` trailing metadata, as it does from the engine.
"""

from __future__ import annotations

import time
from collections.abc import AsyncIterator, Callable, Iterator
from concurrent import futures
from dataclasses import dataclass, field
from datetime import timedelta

import grpc
import pytest
import tenacity
from google.protobuf import any_pb2, duration_pb2
from google.rpc import error_details_pb2, status_pb2  # type: ignore[import-untyped]
from grpc_status import rpc_status
from urllib3 import HTTPHeaderDict

from hatchet_sdk.clients.rest.exceptions import TooManyRequestsException
from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import TenacityConfig

METHOD = "/test.Throttled/Call"


@dataclass
class Throttled:
    """Fails one call per queued (code, RetryInfo delay in seconds or None), then succeeds."""

    failures: list[tuple[grpc.StatusCode, float | None]] = field(default_factory=list)
    calls: int = 0

    def next_failure(self) -> grpc.Status | None:
        self.calls += 1
        if not self.failures:
            return None

        code, delay = self.failures.pop(0)
        status = status_pb2.Status(code=code.value[0], message="throttled")
        if delay is not None:
            retry_delay = duration_pb2.Duration()
            retry_delay.FromTimedelta(timedelta(seconds=delay))
            detail = any_pb2.Any()
            detail.Pack(error_details_pb2.RetryInfo(retry_delay=retry_delay))
            status.details.append(detail)
        return rpc_status.to_status(status)


@pytest.fixture
def throttled() -> Throttled:
    return Throttled()


@pytest.fixture
def channel(throttled: Throttled) -> Iterator[grpc.Channel]:
    def handle(_request: bytes, context: grpc.ServicerContext) -> bytes:
        failure = throttled.next_failure()
        if failure is not None:
            context.abort_with_status(failure)
        return b"ok"

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=2))
    server.add_generic_rpc_handlers(
        (
            grpc.method_handlers_generic_handler(
                "test.Throttled", {"Call": grpc.unary_unary_rpc_method_handler(handle)}
            ),
        )
    )
    port = server.add_insecure_port("localhost:0")
    server.start()
    with grpc.insecure_channel(f"localhost:{port}") as ch:
        yield ch
    server.stop(None)


@pytest.fixture
async def aio_channel(throttled: Throttled) -> AsyncIterator[grpc.aio.Channel]:
    async def handle(
        _request: bytes, context: grpc.aio.ServicerContext[bytes, bytes]
    ) -> bytes:
        failure = throttled.next_failure()
        if failure is not None:
            await context.abort_with_status(failure)
        return b"ok"

    server = grpc.aio.server()
    server.add_generic_rpc_handlers(
        (
            grpc.method_handlers_generic_handler(
                "test.Throttled", {"Call": grpc.unary_unary_rpc_method_handler(handle)}
            ),
        )
    )
    port = server.add_insecure_port("localhost:0")
    await server.start()
    async with grpc.aio.insecure_channel(f"localhost:{port}") as ch:
        yield ch
    await server.stop(None)


def config(
    sleeps: list[float], fixed_wait: float | None = None, max_attempts: int = 5
) -> TenacityConfig:
    """A TenacityConfig that records each wait. With `fixed_wait`, the client's own wait is that constant."""

    def record(retry_state: tenacity.RetryCallState) -> None:
        assert retry_state.next_action is not None
        sleeps.append(retry_state.next_action.sleep)

    if fixed_wait is None:
        return TenacityConfig(max_attempts=max_attempts, before_sleep=record)

    wait = fixed_wait

    class WaitFixed(tenacity.wait_fixed):
        def __init__(self) -> None:
            super().__init__(wait)

    return TenacityConfig(
        max_attempts=max_attempts, before_sleep=record, wait=WaitFixed
    )


def retrying(channel: grpc.Channel, cfg: TenacityConfig) -> Callable[[bytes], bytes]:
    stub: grpc.UnaryUnaryMultiCallable[bytes, bytes] = channel.unary_unary(METHOD)
    return tenacity_retry(stub, cfg)


def test_resource_exhausted_waits_the_server_delay(
    channel: grpc.Channel, throttled: Throttled
) -> None:
    throttled.failures = [(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.3)]
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps))

    start = time.monotonic()
    assert call(b"") == b"ok"
    elapsed = time.monotonic() - start

    # The default backoff would wait at least 1s before the first retry.
    assert sleeps == [pytest.approx(0.3)]
    assert 0.3 <= elapsed < 1.0
    assert throttled.calls == 2


@pytest.mark.asyncio(loop_scope="session")
async def test_aio_resource_exhausted_waits_the_server_delay(
    aio_channel: grpc.aio.Channel, throttled: Throttled
) -> None:
    throttled.failures = [(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.3)]
    sleeps: list[float] = []
    stub: grpc.aio.UnaryUnaryMultiCallable[bytes, bytes] = aio_channel.unary_unary(
        METHOD
    )

    async def call() -> bytes:
        return await stub(b"")

    start = time.monotonic()
    assert await tenacity_retry(call, config(sleeps))() == b"ok"
    elapsed = time.monotonic() - start

    assert sleeps == [pytest.approx(0.3)]
    assert 0.3 <= elapsed < 1.0
    assert throttled.calls == 2


def test_server_delay_is_capped_at_the_client_wait(
    channel: grpc.Channel, throttled: Throttled
) -> None:
    throttled.failures = [(grpc.StatusCode.RESOURCE_EXHAUSTED, 5.0)]
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps, fixed_wait=0.2))

    assert call(b"") == b"ok"
    assert sleeps == [pytest.approx(0.2)]


@pytest.mark.parametrize(
    "code",
    [
        grpc.StatusCode.RESOURCE_EXHAUSTED,
        grpc.StatusCode.UNAVAILABLE,
        grpc.StatusCode.INTERNAL,
    ],
)
def test_errors_without_retry_info_keep_the_client_wait(
    channel: grpc.Channel, throttled: Throttled, code: grpc.StatusCode
) -> None:
    throttled.failures = [(code, None), (code, None)]
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps, fixed_wait=0.2))

    assert call(b"") == b"ok"
    assert sleeps == [pytest.approx(0.2), pytest.approx(0.2)]


def test_server_delay_does_not_carry_over(
    channel: grpc.Channel, throttled: Throttled
) -> None:
    throttled.failures = [
        (grpc.StatusCode.RESOURCE_EXHAUSTED, 0.05),
        (grpc.StatusCode.UNAVAILABLE, None),
    ]
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps, fixed_wait=0.2))

    assert call(b"") == b"ok"
    assert sleeps == [pytest.approx(0.05), pytest.approx(0.2)]


def test_attempt_limit_is_unchanged(
    channel: grpc.Channel, throttled: Throttled
) -> None:
    throttled.failures = [(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.01)] * 5
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps, max_attempts=3))

    with pytest.raises(grpc.RpcError) as exc_info:
        call(b"")

    assert exc_info.value.code() == grpc.StatusCode.RESOURCE_EXHAUSTED
    assert throttled.calls == 3
    assert sleeps == [pytest.approx(0.01), pytest.approx(0.01)]


def test_non_retryable_code_with_retry_info_is_not_retried(
    channel: grpc.Channel, throttled: Throttled
) -> None:
    throttled.failures = [(grpc.StatusCode.INVALID_ARGUMENT, 0.01)]
    sleeps: list[float] = []
    call = retrying(channel, config(sleeps))

    with pytest.raises(grpc.RpcError) as exc_info:
        call(b"")

    assert exc_info.value.code() == grpc.StatusCode.INVALID_ARGUMENT
    assert throttled.calls == 1
    assert sleeps == []


class FakeHttpResponse:
    def __init__(self, headers: HTTPHeaderDict) -> None:
        self.status = 429
        self.reason = "Too Many Requests"
        self.data = b""
        self.headers = headers

    def getheaders(self) -> HTTPHeaderDict:
        return self.headers


@pytest.mark.parametrize(
    ("retry_after", "fixed_wait", "expected"),
    [
        pytest.param("1", 3.0, 1.0, id="waits Retry-After"),
        pytest.param("2", 0.5, 0.5, id="capped at the client wait"),
        pytest.param(
            "Wed, 21 Oct 2015 07:28:00 GMT",
            0.5,
            0.5,
            id="HTTP-date keeps the client wait",
        ),
        pytest.param(None, 0.5, 0.5, id="no Retry-After keeps the client wait"),
    ],
)
def test_rest_429_waits_retry_after(
    retry_after: str | None, fixed_wait: float, expected: float
) -> None:
    headers = HTTPHeaderDict(
        {} if retry_after is None else {"retry-after": retry_after}
    )
    attempts = 0

    def throttled_call() -> str:
        nonlocal attempts
        attempts += 1
        if attempts == 1:
            raise TooManyRequestsException(http_resp=FakeHttpResponse(headers))
        return "ok"

    sleeps: list[float] = []
    cfg = config(sleeps, fixed_wait=fixed_wait).model_copy(update={"retry_429": True})

    assert tenacity_retry(throttled_call, cfg)() == "ok"
    assert sleeps == [pytest.approx(expected)]
