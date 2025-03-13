import asyncio
from contextlib import suppress

from hatchet_sdk import Context, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)

existing_loop_worker = hatchet.workflow(name="WorkerExistingLoopWorkflow")


@existing_loop_worker.task()
async def task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print("started")
    await asyncio.sleep(10)
    print("finished")
    return {"result": "returned result"}


async def async_main() -> None:
    worker = None
    try:
        worker = hatchet.worker(
            "test-worker", slots=1, workflows=[existing_loop_worker]
        )
        worker.start()

        ref = hatchet.admin.run_workflow("MyWorkflow", input={})
        print(await ref.aio_result())
        while True:
            await asyncio.sleep(1)
    finally:
        if worker:
            await worker.exit_gracefully()


def main() -> None:
    with suppress(KeyboardInterrupt):
        asyncio.run(async_main())


if __name__ == "__main__":
    main()
