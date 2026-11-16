from __future__ import annotations

import asyncio

from hatchet_sdk import Context, Hatchet

hatchet = Hatchet()

STREAM_CHUNKS = [f"c{i:02d}|" for i in range(20)]
PRE_STREAM_SLEEP_S = 2.0
INTER_CHUNK_S = 0.05

long_stream = hatchet.workflow(name="subscribe-stream-long")


@long_stream.task()
async def long_streamer(_: None, ctx: Context) -> None:
    await asyncio.sleep(PRE_STREAM_SLEEP_S)
    for chunk in STREAM_CHUNKS:
        await ctx.aio_put_stream(chunk)
        await asyncio.sleep(INTER_CHUNK_S)


# An on_failure handler makes every run a DAG. DAG QUEUED->RUNNING used to be
# published as workflow-run-finished, which hung up stream subscribers as soon
# as the run started.
dag_stream = hatchet.workflow(name="subscribe-stream-dag")


@dag_stream.task()
async def dag_streamer(_: None, ctx: Context) -> None:
    await asyncio.sleep(PRE_STREAM_SLEEP_S)
    for chunk in STREAM_CHUNKS:
        await ctx.aio_put_stream(chunk)


@dag_stream.on_failure_task()
async def on_failure(_: None, _ctx: Context) -> None:
    pass


def main() -> None:
    worker = hatchet.worker(
        "subscribe-stream-dag-worker",
        slots=8,
        workflows=[long_stream, dag_stream],
    )
    worker.start()


if __name__ == "__main__":
    main()
