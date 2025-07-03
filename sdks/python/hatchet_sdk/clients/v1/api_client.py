from collections.abc import Callable
from typing import ParamSpec, TypeVar

import tenacity

from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.configuration import Configuration
from hatchet_sdk.clients.rest.exceptions import ServiceException
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.logger import logger
from hatchet_sdk.utils.typing import JSONSerializableMapping

## Type variables to use with coroutines.
## See https://stackoverflow.com/questions/73240620/the-right-way-to-type-hint-a-coroutine-function
## Return type
R = TypeVar("R")

## Yield type
Y = TypeVar("Y")

## Send type
S = TypeVar("S")

P = ParamSpec("P")


def maybe_additional_metadata_to_kv(
    additional_metadata: dict[str, str] | JSONSerializableMapping | None,
) -> list[str] | None:
    if not additional_metadata:
        return None

    return [f"{k}:{v}" for k, v in additional_metadata.items()]


class BaseRestClient:
    def __init__(self, config: ClientConfig) -> None:
        self.tenant_id = config.tenant_id

        self.client_config = config
        self.api_config = Configuration(
            host=config.server_url,
            access_token=config.token,
        )

        self.api_config.datetime_format = "%Y-%m-%dT%H:%M:%S.%fZ"

    def client(self) -> ApiClient:
        return ApiClient(self.api_config)


def retry(func: Callable[P, R]) -> Callable[P, R]:
    return tenacity.retry(
        reraise=True,
        wait=tenacity.wait_exponential_jitter(),
        stop=tenacity.stop_after_attempt(5),
        before_sleep=_alert_on_retry,
        retry=tenacity.retry_if_exception(_should_retry),
    )(func)


def _alert_on_retry(retry_state: tenacity.RetryCallState) -> None:
    logger.debug(
        f"Retrying {retry_state.fn}: attempt "
        f"{retry_state.attempt_number} ended with: {retry_state.outcome}",
    )


def _should_retry(ex: BaseException) -> bool:
    return isinstance(ex, ServiceException)
