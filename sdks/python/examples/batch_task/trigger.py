from examples.batch_task.worker import BatchInput, workflow
import asyncio

async def main() -> None:
    # Trigger three runs in the same group so the batch flushes at size=3.
    await asyncio.gather(
        workflow.aio_run(BatchInput(message="alpha", group="tenant-1")),
        workflow.aio_run(BatchInput(message="bravo", group="tenant-1")),
        workflow.aio_run(BatchInput(message="charlie", group="tenant-1")),
    )


if __name__ == "__main__":
    asyncio.run(main())
