import asyncio
import time

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

cancellation_workflow = hatchet.workflow(name="CancelWorkflow")


# ❓ Self-cancelling task
@cancellation_workflow.task()
async def self_cancel(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(2)

    ## Cancel the task
    await ctx.aio_cancel()

    await asyncio.sleep(10)

    return {"error": "Task should have been cancelled"}


# !!


# ❓ Checking exit flag
@cancellation_workflow.task()
def check_flag(input: EmptyModel, ctx: Context) -> dict[str, str]:
    for i in range(25):
        time.sleep(1)

        # Note: Checking the status of the exit flag is mostly useful for cancelling
        # sync tasks without needing to forcibly kill the thread they're running on.
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
