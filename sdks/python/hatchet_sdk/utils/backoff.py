import asyncio
import random


async def exp_backoff_sleep(attempt: int, max_sleep_time: float = 5) -> None:
    base_time = 0.1  # starting sleep time in seconds (100 milliseconds)
    jitter = random.uniform(0, base_time)  # add random jitter
    sleep_time = min(base_time * (2**attempt) + jitter, max_sleep_time)
    await asyncio.sleep(sleep_time)
