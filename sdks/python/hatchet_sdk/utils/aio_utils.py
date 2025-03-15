import asyncio
import inspect
from functools import partial
from typing import Awaitable, Callable, ParamSpec, TypeVar

R = TypeVar("R")
P = ParamSpec("P")


## TODO: Stricter typing here
def sync_to_async(func: Callable[P, R]) -> Callable[P, Awaitable[R]]:
    """
    A decorator to run a synchronous function or coroutine in an asynchronous context with added
    asyncio loop safety.

    This decorator allows you to safely call synchronous functions or coroutines from an
    asynchronous function by running them in an executor.

    Args:
        func (callable): The synchronous function or coroutine to be run asynchronously.

    Returns:
        callable: An asynchronous wrapper function that runs the given function in an executor.

    Example:
        @sync_to_async
        def sync_function(x, y):
            return x + y

        async def main():
            result = await sync_function(1, 2)

            print(result)

        asyncio.run(main())
    """

    async def run(*args: P.args, **kwargs: P.kwargs) -> R:
        """
        The asynchronous wrapper function that runs the given function in an executor.

        Args:
            *args: Positional arguments to pass to the function.
            **kwargs: Keyword arguments to pass to the function.

        Returns:
            The result of the function call.
        """
        if inspect.iscoroutinefunction(func):
            # Wrap the coroutine to run it in an executor
            raise TypeError("`func` must be a synchronous function")
        else:
            # Run the synchronous function in an executor
            pfunc = partial(func, *args, **kwargs)
            return await asyncio.to_thread(pfunc)

    return run


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
