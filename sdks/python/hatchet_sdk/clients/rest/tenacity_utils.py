from __future__ import annotations

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
        return tenacity_should_retry(ex, config)

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


def wait_server_retry_delay(
    fallback: Callable[[tenacity.RetryCallState], float],
) -> Callable[[tenacity.RetryCallState], float]:
    """Wait the delay the server asked for when the failed attempt carried one, capped at the
    fallback's wait so a call never waits longer than it would have without it. Otherwise wait
    the fallback's delay.
    """

    def wait(retry_state: tenacity.RetryCallState) -> float:
        fallback_wait = fallback(retry_state)
        assert retry_state.outcome is not None
        server_delay = _server_retry_delay(retry_state.outcome.exception())
        return min(server_delay, fallback_wait) if server_delay > 0 else fallback_wait

    return wait


def _server_retry_delay(ex: BaseException | None) -> float:
    """Seconds to wait from a gRPC error's RetryInfo detail or an HTTP 429's Retry-After header, or 0."""
    if isinstance(ex, TooManyRequestsException):
        retry_after = (ex.headers or {}).get("Retry-After", "")
        return float(retry_after) if retry_after.isdigit() else 0

    if not isinstance(ex, (grpc.Call, grpc.aio.AioRpcError)):
        return 0

    for key, value in ex.trailing_metadata() or ():
        if key == "grpc-status-details-bin":
            for detail in status_pb2.Status.FromString(value).details:
                info = error_details_pb2.RetryInfo()
                if detail.Unpack(info):
                    delay: timedelta = info.retry_delay.ToTimedelta()
                    return delay.total_seconds()

    return 0
