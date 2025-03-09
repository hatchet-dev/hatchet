import asyncio

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

wf = hatchet.workflow(name="CancelWorkflow", on_events=["user:create"])


@wf.task(timeout="10s", retries=1)
async def step1(input: EmptyModel, context: Context) -> None:
    i = 0
    while not context.exit_flag and i < 20:
        print(f"Waiting for cancellation {i}")
        await asyncio.sleep(1)
        i += 1

    if context.exit_flag:
        print("Cancelled")


def main() -> None:
    worker = hatchet.worker("cancellation-worker", max_runs=4, workflows=[wf])

    worker.start()


if __name__ == "__main__":
    main()
