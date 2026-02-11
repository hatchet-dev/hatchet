from __future__ import annotations

import re
from collections.abc import Callable
from typing import TYPE_CHECKING, ParamSpec, TypeVar

import grpc
import tenacity

from hatchet_sdk.clients.rest.exceptions import (
    NotFoundException,
    RestTransportError,
    ServiceException,
)
from hatchet_sdk.logger import logger

if TYPE_CHECKING:
    from hatchet_sdk.config import TenacityConfig

P = ParamSpec("P")
R = TypeVar("R")

# Pattern to extract HTTP method from exception reason
_METHOD_PATTERN = re.compile(r"method=(\w+)", re.IGNORECASE)


def tenacity_retry(func: Callable[P, R], config: TenacityConfig) -> Callable[P, R]:
    if config.max_attempts <= 0:
        return func

    def should_retry(ex: BaseException) -> bool:
        return tenacity_should_retry(ex, config)

    return tenacity.retry(
        reraise=True,
        wait=tenacity.wait_exponential_jitter(),
        stop=tenacity.stop_after_attempt(config.max_attempts),
        before_sleep=tenacity_alert_retry,
        retry=tenacity.retry_if_exception(should_retry),
    )(func)


def tenacity_alert_retry(retry_state: tenacity.RetryCallState) -> None:
    """Called between tenacity retries."""
    logger.debug(
        f"retrying {retry_state.fn}: attempt "
        f"{retry_state.attempt_number} ended with: {retry_state.outcome}",
    )


def tenacity_should_retry(
    ex: BaseException, config: TenacityConfig | None = None
) -> bool:
    """Determine if an exception should trigger a retry.

    Args:
        ex: The exception to evaluate.
        config: Optional tenacity config for transport error settings.

    Returns:
        True if the exception should be retried, False otherwise.
    """
    # HTTP errors: ServiceException (5xx) and NotFoundException (404) are retried
    if isinstance(ex, ServiceException | NotFoundException):
        return True

    # gRPC errors: retry most, except specific permanent failure codes
    if isinstance(ex, grpc.aio.AioRpcError | grpc.RpcError):
        return ex.code() not in [
            grpc.StatusCode.UNIMPLEMENTED,
            grpc.StatusCode.NOT_FOUND,
            grpc.StatusCode.INVALID_ARGUMENT,
            grpc.StatusCode.ALREADY_EXISTS,
            grpc.StatusCode.UNAUTHENTICATED,
            grpc.StatusCode.PERMISSION_DENIED,
        ]

    # REST transport errors: opt-in retry for configured HTTP methods
    if isinstance(ex, RestTransportError):
        if config is not None and config.retry_transport_errors:
            method = _extract_method_from_reason(ex.reason)
            if method is not None:
                allowed_methods = {m.upper() for m in config.retry_transport_methods}
                return method.upper() in allowed_methods
        return False

    return False


def _extract_method_from_reason(reason: str | None) -> str | None:
    """Extract HTTP method from exception reason string.

    The reason string contains 'method=GET' or similar from rest.py exception handling.
    """
    if not reason:
        return None
    match = _METHOD_PATTERN.search(reason)
    return match.group(1) if match else None
