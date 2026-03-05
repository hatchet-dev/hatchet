import asyncio

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    EmptyModel,
    Hatchet,
)

hatchet = Hatchet(debug=True)


# > Step 01 Define Streaming Task
@hatchet.task(
    concurrency=ConcurrencyExpression(
        expression="'constant'",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
    )
)
async def stream_task(input: EmptyModel, ctx: Context) -> dict:
    """Emit chunks to subscribers in real-time."""
    for i in range(5):
        await ctx.aio_put_stream(f"chunk-{i}")
        await asyncio.sleep(0.5)
    return {"status": "done"}




# > Step 02 Emit Chunks
async def _emit_chunks(ctx: Context) -> None:
    for i in range(5):
        await ctx.aio_put_stream(f"chunk-{i}")
        await asyncio.sleep(0.5)


def main() -> None:
    # > Step 04 Run Worker
    worker = hatchet.worker(
        "streaming-worker",
        workflows=[stream_task],
    )
    worker.start()


if __name__ == "__main__":
    main()
