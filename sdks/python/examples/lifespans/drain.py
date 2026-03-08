import asyncio
from collections.abc import AsyncGenerator
from dataclasses import dataclass

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()


class FakePool:
    def __init__(self) -> None:
        self._closed = False

    def acquire(self) -> None:
        if self._closed:
            raise RuntimeError("the pool is already closed")

    def close(self) -> None:
        self._closed = True

    @property
    def closed(self) -> bool:
        return self._closed


async def drain_lifespan() -> AsyncGenerator[FakePool, None]:
    pool = FakePool()
    try:
        yield pool
    finally:
        pool.close()


@dataclass
class DrainResult:
    status: str
    iterations: int


@dataclass
class DrainInput:
    n: int


@hatchet.task(name="LifespanDrainTask", input_validator=DrainInput)
async def lifespan_drain_task(input: DrainInput, ctx: Context) -> DrainResult:
    pool: FakePool = ctx.lifespan
    iters = 0
    for _ in range(input.n):
        pool.acquire()
        iters += 1
        await asyncio.sleep(1)

    return DrainResult(status="ok", iterations=iters)


def main() -> None:
    worker = hatchet.worker(
        "drain-test-worker",
        slots=5,
        workflows=[lifespan_drain_task],
        lifespan=drain_lifespan,
    )
    worker.start()


if __name__ == "__main__":
    main()
