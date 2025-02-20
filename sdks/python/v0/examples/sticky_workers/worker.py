from dotenv import load_dotenv

from hatchet_sdk import Context, Hatchet, StickyStrategy
from hatchet_sdk.context.context import ContextAioImpl

load_dotenv()

hatchet = Hatchet(debug=True)


@hatchet.workflow(on_events=["sticky:parent"], sticky=StickyStrategy.SOFT)
class StickyWorkflow:
    @hatchet.step()
    def step1a(self, context: Context) -> dict[str, str | None]:
        return {"worker": context.worker.id()}

    @hatchet.step()
    def step1b(self, context: Context) -> dict[str, str | None]:
        return {"worker": context.worker.id()}

    @hatchet.step(parents=["step1a", "step1b"])
    async def step2(self, context: ContextAioImpl) -> dict[str, str | None]:
        ref = await context.spawn_workflow(
            "StickyChildWorkflow", {}, options={"sticky": True}
        )

        await ref.result()

        return {"worker": context.worker.id()}


@hatchet.workflow(on_events=["sticky:child"], sticky=StickyStrategy.SOFT)
class StickyChildWorkflow:
    @hatchet.step()
    def child(self, context: Context) -> dict[str, str | None]:
        return {"worker": context.worker.id()}


def main() -> None:
    worker = hatchet.worker("sticky-worker", max_runs=10)
    worker.register_workflow(StickyWorkflow())
    worker.register_workflow(StickyChildWorkflow())
    worker.start()


if __name__ == "__main__":
    main()
