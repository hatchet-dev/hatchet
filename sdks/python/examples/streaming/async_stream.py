from examples.streaming.worker import streaming_workflow


async def main() -> None:
    ref = await streaming_workflow.aio_run_no_wait()

    stream = ref.stream()

    async for chunk in stream:
        print(chunk)


if __name__ == "__main__":
    import asyncio

    asyncio.run(main())
