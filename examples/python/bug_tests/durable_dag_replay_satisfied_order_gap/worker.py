import asyncio

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet()

replay_gap = hatchet.workflow(name="durable-dag-replay-gap")


@replay_gap.task()
async def quick(_: EmptyModel, ctx: Context) -> dict[str, int]:
    await asyncio.sleep(0.5)
    return {"retry_count": ctx.retry_count}


@replay_gap.task()
async def flaky(_: EmptyModel, ctx: Context) -> dict[str, int]:
    await asyncio.sleep(1.5)

    if ctx.retry_count == 0:
        raise ValueError("fails on the first attempt and succeeds when replayed")

    return {"retry_count": ctx.retry_count}


@replay_gap.task()
async def slow(_: EmptyModel, ctx: Context) -> dict[str, int]:
    await asyncio.sleep(5)
    return {"retry_count": ctx.retry_count}
