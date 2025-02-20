import asyncio

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["async:create"])
class AsyncWorkflow:

    @hatchet.step(timeout="10s")
    async def step1(self, context: Context) -> dict[str, str]:
        print("started step1")
        return {"test": "test"}

    @hatchet.step(parents=["step1"], timeout="10s")
    async def step2(self, context: Context) -> None:
        print("finished step2")


async def _main() -> None:
    workflow = AsyncWorkflow()
    worker = hatchet.worker("async-worker", max_runs=4)
    worker.register_workflow(workflow)
    await worker.async_start()


def main() -> None:
    asyncio.run(_main())


if __name__ == "__main__":
    main()
