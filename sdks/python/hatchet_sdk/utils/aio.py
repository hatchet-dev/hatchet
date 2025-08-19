import asyncio
from collections.abc import Coroutine
from typing import TypeVar

T = TypeVar("T")


async def gather_max_concurrency(
    *tasks: Coroutine[None, None, T],
    max_concurrency: int,
) -> list[T]:
    sem = asyncio.Semaphore(max_concurrency)

    async def task_wrapper(task: Coroutine[None, None, T]) -> T:
        async with sem:
            return await task

    return await asyncio.gather(
        *(task_wrapper(task) for task in tasks),
        return_exceptions=False,
    )
