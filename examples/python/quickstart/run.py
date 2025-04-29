import asyncio

from .workflows.first_task import SimpleInput, first_task


async def main() -> None:
    result = await first_task.aio_run(SimpleInput(message="Hello World!"))

    print(
        "Finished running task, and got the transformed message! The transformed message is:",
        result.transformed_message,
    )


if __name__ == "__main__":
    asyncio.run(main())
