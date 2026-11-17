"""Unit tests for retries that wait the delay a server asks for in a google.rpc.RetryInfo detail."""

import json
import random
import time
from collections.abc import Callable, Iterator
from concurrent import futures
from contextlib import contextmanager
from datetime import timedelta
from pathlib import Path
from typing import Any

import grpc
import pytest
import tenacity
from google.protobuf import duration_pb2
from google.rpc import error_details_pb2, status_pb2  # type: ignore[import-untyped]
from grpc_status import rpc_status

from hatchet_sdk.clients.rest.tenacity_utils import tenacity_retry
from hatchet_sdk.config import TenacityConfig

OWN_WAIT = 0.2

THROTTLED_CALLS: dict[str, Any] = json.loads(
    (
        Path(__file__).resolve().parents[4]
        / "api-contracts/sdk-conformance/throttled_calls.json"
    ).read_text()
)
SCENARIOS: list[dict[str, Any]] = THROTTLED_CALLS["scenarios"]
JITTER: float = THROTTLED_CALLS["jitter"]


class OwnWait(tenacity.wait_fixed):
    """The client's own wait, a constant that differs from every server delay in these tests."""

    def __init__(self) -> None:
        super().__init__(OWN_WAIT)


def failure(code: grpc.StatusCode, retry_delay: float | None) -> status_pb2.Status:
    """A failed call's status, with `retry_delay` seconds in a RetryInfo detail unless it is None."""
    status = status_pb2.Status(code=code.value[0], message="throttled")
    if retry_delay is not None:
        delay = duration_pb2.Duration()
        delay.FromTimedelta(timedelta(seconds=retry_delay))
        status.details.add().Pack(error_details_pb2.RetryInfo(retry_delay=delay))
    return status


def async_error(retry_delay: float) -> grpc.aio.AioRpcError:
    status = failure(grpc.StatusCode.RESOURCE_EXHAUSTED, retry_delay)
    return grpc.aio.AioRpcError(
        grpc.StatusCode.RESOURCE_EXHAUSTED,
        grpc.aio.Metadata(),
        grpc.aio.Metadata(("grpc-status-details-bin", status.SerializeToString())),
    )


def recording(waits: list[float]) -> Callable[[tenacity.RetryCallState], None]:
    """A before_sleep hook that appends each wait tenacity chose to `waits`."""

    def record(retry_state: tenacity.RetryCallState) -> None:
        assert retry_state.next_action is not None
        waits.append(retry_state.next_action.sleep)

    return record


@contextmanager
def throttling_server(failures: list[status_pb2.Status]) -> Iterator[int]:
    """Serves /test.Throttled/Call, which fails with each of `failures` in turn and then answers
    b"ok". Yields the port."""

    def handle(_request: bytes, context: grpc.ServicerContext) -> bytes:
        if failures:
            context.abort_with_status(rpc_status.to_status(failures.pop(0)))
        return b"ok"

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=1))
    server.add_generic_rpc_handlers(
        (
            grpc.method_handlers_generic_handler(
                "test.Throttled", {"Call": grpc.unary_unary_rpc_method_handler(handle)}
            ),
        )
    )
    port = server.add_insecure_port("localhost:0")
    server.start()
    try:
        yield port
    finally:
        server.stop(None)


@pytest.fixture(autouse=True)
def shortest_waits(monkeypatch: pytest.MonkeyPatch) -> None:
    """Holds the random source at its lowest value, so a wait on a server delay is not extended."""
    monkeypatch.setattr(random, "uniform", lambda low, _high: low)


@pytest.mark.parametrize("scenario", SCENARIOS, ids=[s["name"] for s in SCENARIOS])
@pytest.mark.parametrize("longest_waits", [False, True], ids=["shortest", "longest"])
def test_throttled_call_scenario(
    scenario: dict[str, Any], longest_waits: bool, monkeypatch: pytest.MonkeyPatch
) -> None:
    """Runs a scenario that every Hatchet SDK's tests share, with the default attempt limit and
    the random source held at its lowest or its highest value."""
    monkeypatch.setattr(time, "sleep", lambda _seconds: None)
    if longest_waits:
        monkeypatch.setattr(random, "uniform", lambda _low, high: high)
    extend = 1 + JITTER if longest_waits else 1
    delays = [delay_ms / 1000 for delay_ms in scenario["delays_ms"]]
    errors: list[grpc.aio.AioRpcError] = []
    calls = 0

    def call() -> str:
        nonlocal calls
        calls += 1
        if len(errors) < len(delays):
            errors.append(async_error(delays[len(errors)]))
            raise errors[-1]
        return "ok"

    waits: list[float] = []
    retried = tenacity_retry(call, TenacityConfig(before_sleep=recording(waits)))

    if scenario["succeeds"]:
        assert retried() == "ok"
    else:
        with pytest.raises(grpc.aio.AioRpcError) as raised:
            retried()
        assert raised.value is errors[-1]

    assert waits == [
        pytest.approx(wait_ms / 1000 * extend) for wait_ms in scenario["waits_ms"]
    ]
    assert calls == len(waits) + 1


def test_retry_waits_the_server_delay_of_the_failed_attempt(
    monkeypatch: pytest.MonkeyPatch,
) -> None:
    """Runs against a real gRPC server, so the errors are the ones grpcio raises. The random
    source is held at its highest value, so only the wait on a server delay is extended.
    """
    monkeypatch.setattr(random, "uniform", lambda _low, high: high)
    failures = [
        failure(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.3),
        failure(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.0),
        failure(grpc.StatusCode.UNAVAILABLE, None),
    ]
    waits: list[float] = []

    with (
        throttling_server(failures) as port,
        grpc.insecure_channel(f"localhost:{port}") as channel,
    ):
        stub: grpc.UnaryUnaryMultiCallable[bytes, bytes] = channel.unary_unary(
            "/test.Throttled/Call"
        )
        call = tenacity_retry(
            stub, TenacityConfig(before_sleep=recording(waits), wait=OwnWait)
        )
        assert call(b"") == b"ok"

    # The server's delay, then the client's own wait after a delay of zero and after an error
    # without one: a delay is used once and does not carry over.
    assert waits == [pytest.approx(0.3 * (1 + JITTER)), OWN_WAIT, OWN_WAIT]


async def test_async_call_waits_the_server_delay() -> None:
    """Runs against a real gRPC server through an aio channel, so the error is the one grpc.aio
    raises."""
    waits: list[float] = []

    with throttling_server([failure(grpc.StatusCode.RESOURCE_EXHAUSTED, 0.3)]) as port:
        async with grpc.aio.insecure_channel(f"localhost:{port}") as channel:
            stub: grpc.aio.UnaryUnaryMultiCallable[bytes, bytes] = channel.unary_unary(
                "/test.Throttled/Call"
            )

            async def call() -> bytes:
                return await stub(b"")

            retried = tenacity_retry(
                call, TenacityConfig(before_sleep=recording(waits), wait=OwnWait)
            )
            assert await retried() == b"ok"

    assert waits == [pytest.approx(0.3)]
