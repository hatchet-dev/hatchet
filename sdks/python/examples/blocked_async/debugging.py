# > Functions
import asyncio
import time

sleep_time = 3

async def blocking() -> None:
    for i in range(sleep_time):
        print("Blocking", i)
        time.sleep(1)

async def non_blocking(task_id: str = "Non-blocking") -> None:
    for i in range(sleep_time):
        print(task_id, i)
        await asyncio.sleep(1)
# !!

# > Blocked
async def blocked() -> None:
    loop = asyncio.get_event_loop()

    await asyncio.gather(*[
        loop.create_task(blocking()),
        loop.create_task(non_blocking()),
    ])
# !!

# > Unblocked
async def working() -> None:
    loop = asyncio.get_event_loop()

    await asyncio.gather(*[
        loop.create_task(non_blocking("A")),
        loop.create_task(non_blocking("B")),
    ])
# !!


if __name__ == "__main__":
    asyncio.run(blocked())
    asyncio.run(working())
# !!
