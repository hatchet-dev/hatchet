from __future__ import annotations

import random
from typing import TYPE_CHECKING, ParamSpec, TypeVar

import grpc
import tenacity
from google.rpc import (  # type: ignore[import-untyped]
    error_details_pb2,
    status_pb2,
)

from hatchet_sdk.clients.rest.exceptions import (
    NotFoundError,
    RestTransportError,
    ServiceException,
    TooManyRequestsException,
)

if TYPE_CHECKING:
    from collections.abc import Callable
    from datetime import timedelta

    from hatchet_sdk.config import TenacityConfig

P = ParamSpec("P")
R = TypeVar("R")


def tenacity_retry(func: Callable[P, R], config: TenacityConfig) -> Callable[P, R]:
    if config.max_attempts <= 0:
        return func

    def should_retry(ex: BaseException) -> bool:
        return (
            tenacity_should_retry(ex, config)
            and _server_retry_delay(ex) <= MAX_SERVER_RETRY_DELAY_SECONDS
        )

    return tenacity.retry(
        reraise=True,
        wait=wait_server_retry_delay(config.wait()),
        stop=tenacity.stop_after_attempt(config.max_attempts),
        before_sleep=config.before_sleep,
        retry=tenacity.retry_if_exception(should_retry),
    )(func)


def tenacity_should_retry(
    ex: BaseException, config: TenacityConfig | None = None
) -> bool:
    """Return True when the exception should be retried."""
    if isinstance(ex, (ServiceException, NotFoundError)):
        return True

    if isinstance(ex, TooManyRequestsException):
        return bool(config and config.retry_429)

    # gRPC errors: retry most, except specific permanent failure codes
    if isinstance(ex, (grpc.aio.AioRpcError, grpc.RpcError)):
        non_retryable = [
            grpc.StatusCode.UNIMPLEMENTED,
            grpc.StatusCode.INVALID_ARGUMENT,
            grpc.StatusCode.ALREADY_EXISTS,
            grpc.StatusCode.UNAUTHENTICATED,
            grpc.StatusCode.PERMISSION_DENIED,
        ]
        if not config or not config.retry_not_found:
            ## don't retry NOT_FOUND by default,
            ## but allow it to be configurable so that we can
            ## allow this internally, e.g. in `get_details`
            non_retryable.append(grpc.StatusCode.NOT_FOUND)

        return ex.code() not in non_retryable

    # REST transport errors: opt-in retry for configured HTTP methods
    if isinstance(ex, RestTransportError):
        if config is not None and config.retry_transport_errors:
            method = ex.http_method
            if method is not None:
                return method in config.retry_transport_methods
        return False

    return False


MAX_SERVER_RETRY_DELAY_SECONDS = 5.0
"""The longest wait the SDK accepts from a server's RetryInfo delay. The wait blocks the caller,
so tenacity_retry fails a call at once with the server's error when the server asks for longer."""


SERVER_RETRY_DELAY_JITTER = 0.2
"""The most a wait on a server's RetryInfo delay is extended, as a share of the wait. Clients
throttled at the same moment get the same delay; the random extension keeps them from all coming
back at the same moment. It only extends the wait, since a retry before the server's delay would
be refused."""


def wait_server_retry_delay(
    fallback: Callable[[tenacity.RetryCallState], float],
) -> Callable[[tenacity.RetryCallState], float]:
    """Wait the fallback's delay, unless the failed attempt carried a RetryInfo delay. Then wait
    that delay, doubled for each retry the call has already made and capped at
    MAX_SERVER_RETRY_DELAY_SECONDS, which is the client behavior RetryInfo documents, and extended
    by up to SERVER_RETRY_DELAY_JITTER. A server that keeps throttling repeats the same delay, so
    without the doubling the client would not back off.
    """

    def wait(retry_state: tenacity.RetryCallState) -> float:
        assert retry_state.outcome is not None
        server_delay = _server_retry_delay(retry_state.outcome.exception())
        if server_delay > 0:
            delay = server_delay
            for _ in range(retry_state.attempt_number - 1):
                delay = min(delay * 2, MAX_SERVER_RETRY_DELAY_SECONDS)

            return delay * (1 + random.uniform(0, SERVER_RETRY_DELAY_JITTER))

        # Tenacity's wait strategies keep no state between calls; each wait is computed from
        # retry_state.attempt_number, so skipping the fallback here leaves later waits unchanged.
        return fallback(retry_state)

    return wait


def _server_retry_delay(ex: BaseException | None) -> float:
    """Seconds to wait from a gRPC error's RetryInfo detail, or 0."""
    if not isinstance(ex, (grpc.Call, grpc.aio.AioRpcError)):
        return 0

    for key, value in ex.trailing_metadata() or ():
        if key == "grpc-status-details-bin" and isinstance(value, bytes):
            for detail in status_pb2.Status.FromString(value).details:
                info = error_details_pb2.RetryInfo()
                if detail.Unpack(info):
                    delay: timedelta = info.retry_delay.ToTimedelta()
                    return delay.total_seconds()

    return 0
