import asyncio

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["user:create"])
class CancelWorkflow:
    def __init__(self):
        self.my_value = "test"

    @hatchet.step(timeout="10s", retries=1)
    async def step1(self, context: Context):
        i = 0
        while not context.exit_flag.is_set() and i < 20:
            print(f"Waiting for cancellation {i}")
            await asyncio.sleep(1)
            i += 1

        if context.exit_flag.is_set():
            print("Cancelled")


workflow = CancelWorkflow()
worker = hatchet.worker("test-worker", max_runs=4)
worker.register_workflow(workflow)

worker.start()
