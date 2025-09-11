from typing import Callable, ParamSpec, TypeVar

import grpc
import tenacity

from hatchet_sdk.logger import logger

P = ParamSpec("P")
R = TypeVar("R")


def tenacity_retry(func: Callable[P, R]) -> Callable[P, R]:
    return tenacity.retry(
        reraise=True,
        wait=tenacity.wait_exponential_jitter(),
        stop=tenacity.stop_after_attempt(5),
        before_sleep=tenacity_alert_retry,
        retry=tenacity.retry_if_exception(tenacity_should_retry),
    )(func)


def tenacity_alert_retry(retry_state: tenacity.RetryCallState) -> None:
    """Called between tenacity retries."""
    logger.debug(
        f"Retrying {retry_state.fn}: attempt "
        f"{retry_state.attempt_number} ended with: {retry_state.outcome}",
    )


def tenacity_should_retry(ex: Exception) -> bool:
    if isinstance(ex, (grpc.aio.AioRpcError, grpc.RpcError)):
        if ex.code() in [
            grpc.StatusCode.UNIMPLEMENTED,
            grpc.StatusCode.NOT_FOUND,
        ]:
            return False
        return True
    else:
        return False
