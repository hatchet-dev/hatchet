import asyncio

from hatchet_sdk import BaseWorkflow, Context, Hatchet

hatchet = Hatchet(debug=True)

wf = hatchet.declare_workflow(on_events=["user:create"])


class CancelWorkflow(BaseWorkflow):
    config = wf.config

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
    worker = hatchet.worker("cancellation-worker", max_runs=4)
    worker.register_workflow(CancelWorkflow())

    worker.start()


if __name__ == "__main__":
    main()
