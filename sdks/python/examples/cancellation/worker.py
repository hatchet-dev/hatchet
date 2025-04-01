import asyncio

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

cancellation_workflow = hatchet.workflow(name="CancelWorkflow")


@cancellation_workflow.task()
async def step1(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(2)

    await ctx.aio_cancel()

    await asyncio.sleep(10)

    return {"error": "Task should have been cancelled"}


def main() -> None:
    worker = hatchet.worker("cancellation-worker", workflows=[cancellation_workflow])
    worker.start()


if __name__ == "__main__":
    main()
