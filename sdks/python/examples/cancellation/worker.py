import asyncio
from datetime import timedelta

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(name="CancelWorkflow")


@wf.task(execution_timeout=timedelta(seconds=10), retries=1)
async def step1(input: EmptyModel, ctx: Context) -> None:
    i = 0
    while not ctx.exit_flag and i < 40:
        print(f"Waiting for cancellation {i}")
        await asyncio.sleep(1)
        i += 1

    if ctx.exit_flag:
        print("Cancelled")


def main() -> None:
    worker = hatchet.worker("cancellation-worker", slots=4, workflows=[wf])
    worker.start()


if __name__ == "__main__":
    main()
