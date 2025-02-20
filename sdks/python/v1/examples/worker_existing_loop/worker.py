import asyncio
from contextlib import suppress

from hatchet_sdk import Context, Hatchet
from hatchet_sdk.workflow import BaseWorkflow

hatchet = Hatchet(debug=True)


class MyWorkflow(BaseWorkflow):
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
