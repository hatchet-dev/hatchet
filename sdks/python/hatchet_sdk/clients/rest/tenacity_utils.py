from collections.abc import Callable
from typing import ParamSpec, TypeVar

import grpc
import tenacity

from hatchet_sdk.clients.rest.exceptions import NotFoundException, ServiceException
from hatchet_sdk.config import TenacityConfig
from hatchet_sdk.logger import logger

P = ParamSpec("P")
R = TypeVar("R")


def tenacity_retry(func: Callable[P, R], config: TenacityConfig) -> Callable[P, R]:
    if config.max_attempts <= 0:
        return func

    return tenacity.retry(
        reraise=True,
        wait=tenacity.wait_exponential_jitter(),
        stop=tenacity.stop_after_attempt(config.max_attempts),
        before_sleep=tenacity_alert_retry,
        retry=tenacity.retry_if_exception(tenacity_should_retry),
    )(func)


def tenacity_alert_retry(retry_state: tenacity.RetryCallState) -> None:
    """Called between tenacity retries."""
    logger.debug(
        f"retrying {retry_state.fn}: attempt "
        f"{retry_state.attempt_number} ended with: {retry_state.outcome}",
    )


def tenacity_should_retry(ex: BaseException) -> bool:
    if isinstance(ex, ServiceException | NotFoundException):
        return True

    if isinstance(ex, grpc.aio.AioRpcError | grpc.RpcError):
        return ex.code() not in [
            grpc.StatusCode.UNIMPLEMENTED,
            grpc.StatusCode.NOT_FOUND,
            grpc.StatusCode.INVALID_ARGUMENT,
            grpc.StatusCode.ALREADY_EXISTS,
            grpc.StatusCode.UNAUTHENTICATED,
            grpc.StatusCode.PERMISSION_DENIED,
        ]

    return False
