import asyncio
from concurrent.futures import ThreadPoolExecutor
from typing import Callable, Coroutine, ParamSpec, TypeVar

from hatchet_sdk.clients.rest.configuration import Configuration

## Type variables to use with coroutines.
## See https://stackoverflow.com/questions/73240620/the-right-way-to-type-hint-a-coroutine-function
## Return type
R = TypeVar("R")

## Yield type
Y = TypeVar("Y")

## Send type
S = TypeVar("S")

P = ParamSpec("P")


class BaseRestClient:
    def __init__(self, host: str, api_key: str, tenant_id: str):
        self.tenant_id = tenant_id

        self.config = Configuration(
            host=host,
            access_token=api_key,
        )

    def _run_async_function(
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
                    lambda: self._run_async_function(async_func, *args, **kwargs)
                )
                return future.result()
