import asyncio
from collections.abc import Coroutine
from typing import Literal, TypeVar, overload

T = TypeVar("T")


@overload
async def gather_max_concurrency(
    *tasks: Coroutine[None, None, T],
    max_concurrency: int,
    return_exceptions: Literal[True],
) -> list[T | BaseException]: ...


@overload
async def gather_max_concurrency(
    *tasks: Coroutine[None, None, T],
    max_concurrency: int,
    return_exceptions: Literal[False] = False,
) -> list[T]: ...


async def gather_max_concurrency(
    *tasks: Coroutine[None, None, T],
    max_concurrency: int,
    return_exceptions: bool = False,
) -> list[T] | list[T | BaseException]:
    sem = asyncio.Semaphore(max_concurrency)

    async def task_wrapper(task: Coroutine[None, None, T]) -> T:
        async with sem:
            return await task

    return await asyncio.gather(
        *(task_wrapper(task) for task in tasks),
        return_exceptions=return_exceptions,
    )
