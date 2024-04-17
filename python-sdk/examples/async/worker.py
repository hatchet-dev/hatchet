import asyncio

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["user:create"])
class AsyncWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step(timeout="5s")
    def step1(self, context: Context):
        async def async_step1():
            print("started step1")
            await asyncio.sleep(2)
            print("finished step1")
            return "result"

        res = asyncio.run(async_step1())
        print(res)
        return {"test": "test"}

    @hatchet.step(parents=["step1"], timeout="4s")
    async def step2(self, context):
        print("started async step2")
        await asyncio.sleep(2)
        print("finished step2")


workflow = AsyncWorkflow()
worker = hatchet.worker("test-worker", max_runs=4)
worker.register_workflow(workflow)

worker.start()
