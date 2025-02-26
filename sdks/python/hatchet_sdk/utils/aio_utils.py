import asyncio
import inspect
from concurrent.futures import Executor
from functools import partial, wraps
from typing import Any


## TODO: Stricter typing here
def sync_to_async(func: Any) -> Any:
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

        @sync_to_async
        async def async_function(x, y):
            return x + y


        def undecorated_function(x, y):
            return x + y

        async def main():
            result1 = await sync_function(1, 2)
            result2 = await async_function(3, 4)
            result3 = await sync_to_async(undecorated_function)(5, 6)
            print(result1, result2, result3)

        asyncio.run(main())
    """

    ## TODO: Stricter typing here
    @wraps(func)
    async def run(
        *args: Any,
        loop: asyncio.AbstractEventLoop | None = None,
        executor: Executor | None = None,
        **kwargs: Any
    ) -> Any:
        """
        The asynchronous wrapper function that runs the given function in an executor.

        Args:
            *args: Positional arguments to pass to the function.
            loop (asyncio.AbstractEventLoop, optional): The event loop to use. If None, the current running loop is used.
            executor (concurrent.futures.Executor, optional): The executor to use. If None, the default executor is used.
            **kwargs: Keyword arguments to pass to the function.

        Returns:
            The result of the function call.
        """
        if loop is None:
            loop = asyncio.get_running_loop()

        if inspect.iscoroutinefunction(func):
            # Wrap the coroutine to run it in an executor
            async def wrapper() -> Any:
                return await func(*args, **kwargs)

            pfunc = partial(asyncio.run, wrapper())
            return await loop.run_in_executor(executor, pfunc)
        else:
            # Run the synchronous function in an executor
            pfunc = partial(func, *args, **kwargs)
            return await loop.run_in_executor(executor, pfunc)

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
