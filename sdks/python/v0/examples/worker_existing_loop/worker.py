import asyncio
from contextlib import suppress

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(name="MyWorkflow")
class MyWorkflow:
    @hatchet.step()
    async def step(self, context: Context) -> dict[str, str]:
        print("started")
        await asyncio.sleep(10)
        print("finished")
        return {"result": "returned result"}


async def async_main() -> None:
    worker = None
    try:
        workflow = MyWorkflow()
        worker = hatchet.worker("test-worker", max_runs=1)
        worker.register_workflow(workflow)
        worker.start()

        ref = hatchet.admin.run_workflow("MyWorkflow", input={})
        print(await ref.result())
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
