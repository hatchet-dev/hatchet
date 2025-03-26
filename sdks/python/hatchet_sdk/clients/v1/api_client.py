import asyncio
from concurrent.futures import ThreadPoolExecutor
from typing import AsyncContextManager, Callable, Coroutine, ParamSpec, TypeVar

from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.configuration import Configuration
from hatchet_sdk.config import ClientConfig
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
    additional_metadata: dict[str, str] | JSONSerializableMapping | None
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

    def client(self) -> AsyncContextManager[ApiClient]:
        return ApiClient(self.api_config)

    def _run_async_function_do_not_use_directly(
        self,
        async_func: Callable[P, Coroutine[Y, S, R]],
        *args: P.args,
        **kwargs: P.kwargs,
    ) -> R:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
        try:
            return loop.run_until_complete(async_func(*args, **kwargs))
        finally:
            loop.close()

    def _run_async_from_sync(
        self,
        async_func: Callable[P, Coroutine[Y, S, R]],
        *args: P.args,
        **kwargs: P.kwargs,
    ) -> R:
        try:
            loop = asyncio.get_event_loop()
        except RuntimeError:
            loop = None

        if loop and loop.is_running():
            return loop.run_until_complete(async_func(*args, **kwargs))
        else:
            with ThreadPoolExecutor() as executor:
                future = executor.submit(
                    lambda: self._run_async_function_do_not_use_directly(
                        async_func, *args, **kwargs
                    )
                )
                return future.result()
