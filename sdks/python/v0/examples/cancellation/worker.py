import asyncio

from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["user:create"])
class CancelWorkflow:
    @hatchet.step(timeout="10s", retries=1)
    async def step1(self, context: Context) -> None:
        i = 0
        while not context.exit_flag and i < 20:
            print(f"Waiting for cancellation {i}")
            await asyncio.sleep(1)
            i += 1

        if context.exit_flag:
            print("Cancelled")


def main() -> None:
    workflow = CancelWorkflow()
    worker = hatchet.worker("cancellation-worker", max_runs=4)
    worker.register_workflow(workflow)

    worker.start()


if __name__ == "__main__":
    main()
