from __future__ import annotations

from typing import TYPE_CHECKING, ParamSpec, TypeVar

import grpc
import tenacity

from hatchet_sdk.clients.rest.exceptions import (
    NotFoundException,
    RestTransportError,
    ServiceException,
    TooManyRequestsException,
)

if TYPE_CHECKING:
    from collections.abc import Callable

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
        wait=config.wait(),
        stop=tenacity.stop_after_attempt(config.max_attempts),
        before_sleep=config.before_sleep,
        retry=tenacity.retry_if_exception(should_retry),
    )(func)


def tenacity_should_retry(
    ex: BaseException, config: TenacityConfig | None = None
) -> bool:
    """Return True when the exception should be retried."""
    if isinstance(ex, (ServiceException, NotFoundException)):
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
