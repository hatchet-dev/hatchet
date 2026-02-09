import asyncio

from workflows.first_workflow import my_task


async def main() -> None:
    result = await my_task.aio_run()

    print(
        "Finished running task, and got the meaning of life! The meaning of life is:",
        result["meaning_of_life"],
    )


if __name__ == "__main__":
    asyncio.run(main())
