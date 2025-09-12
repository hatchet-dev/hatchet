import asyncio
import time


# > Async-safe
async def async_safe() -> int:
    await asyncio.sleep(5)

    return 42




# > Blocking
async def blocking() -> int:
    time.sleep(5)

    return 42




# > Using to_thread
async def to_thread() -> int:
    await asyncio.to_thread(time.sleep, 5)

    return 42




# > Using run_in_executor
async def run_in_executor() -> int:
    loop = asyncio.get_event_loop()

    await loop.run_in_executor(None, time.sleep, 5)

    return 42


