import asyncio
import time

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

cancellation_workflow = hatchet.workflow(name="CancelWorkflow")


# > Async cancellation
@cancellation_workflow.task()
async def async_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    try:
        await asyncio.sleep(60)
    except asyncio.CancelledError:
        print("Cleaning up resources...")
        raise

    return {"status": "completed"}


# !!


# > Self-cancelling task
@cancellation_workflow.task()
async def self_cancel(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(2)

    ## Cancel the task
    await ctx.aio_cancel()

    await asyncio.sleep(10)

    return {"error": "Task should have been cancelled"}


# !!


# > Checking exit flag
@cancellation_workflow.task()
def check_flag(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(3):
        time.sleep(1)

        if ctx.exit_flag:
            print("Task has been cancelled")
            raise ValueError("Task has been cancelled")

    return {"error": "Task should have been cancelled"}


# !!


def main() -> None:
    worker = hatchet.worker("cancellation-worker", workflows=[cancellation_workflow])
    worker.start()


if __name__ == "__main__":
    main()
