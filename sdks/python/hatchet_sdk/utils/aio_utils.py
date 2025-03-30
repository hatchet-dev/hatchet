import asyncio
from concurrent.futures import ThreadPoolExecutor
from typing import Callable, Coroutine, ParamSpec, TypeVar

P = ParamSpec("P")
R = TypeVar("R")
Y = TypeVar("Y")
S = TypeVar("S")


def get_active_event_loop() -> asyncio.AbstractEventLoop | None:
    """
    Get the active event loop.

    Returns:
        asyncio.AbstractEventLoop: The active event loop, or None if there is no active
        event loop in the current thread.
    """
    try:
        return asyncio.get_event_loop()
    except RuntimeError as e:
        if str(e).startswith("There is no current event loop in thread"):
            return None
        else:
            raise e


def _run_async_function_do_not_use_directly(
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


def run_async_from_sync(
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
                lambda: _run_async_function_do_not_use_directly(
                    async_func, *args, **kwargs
                )
            )
            return future.result()
