# > Worker
import asyncio
import time

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

SLEEP_TIME = 6


@hatchet.task()
async def non_blocking_async(input: None, ctx: Context) -> None:
    for i in range(SLEEP_TIME):
        print("Non blocking async", i)
        await asyncio.sleep(1)


@hatchet.task()
def non_blocking_sync(input: None, ctx: Context) -> None:
    for i in range(SLEEP_TIME):
        print("Non blocking sync", i)
        time.sleep(1)


@hatchet.task()
async def blocking(input: None, ctx: Context) -> None:
    for i in range(SLEEP_TIME):
        print("Blocking", i)
        time.sleep(1)


def main() -> None:
    worker = hatchet.worker(
        "test-worker", workflows=[non_blocking_async, non_blocking_sync, blocking]
    )
    worker.start()


if __name__ == "__main__":
    main()
