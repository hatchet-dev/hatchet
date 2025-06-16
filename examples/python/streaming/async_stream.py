import asyncio

from examples.streaming.worker import stream_task


async def main() -> None:
    ref = await stream_task.aio_run_no_wait()
    await asyncio.sleep(1)

    stream = ref._wrr.stream()

    async for chunk in stream:
        print(chunk)


if __name__ == "__main__":
    import asyncio

    asyncio.run(main())
