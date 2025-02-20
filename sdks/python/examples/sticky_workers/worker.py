from hatchet_sdk import (
    BaseWorkflow,
    ChildTriggerWorkflowOptions,
    Context,
    Hatchet,
    StickyStrategy,
)

hatchet = Hatchet(debug=True)

sticky_workflow = hatchet.declare_workflow(
    on_events=["sticky:parent"], sticky=StickyStrategy.SOFT
)


class StickyWorkflow(BaseWorkflow):
    config = sticky_workflow.config

    @hatchet.step()
    def step1a(self, context: Context) -> dict[str, str | None]:
        return {"worker": context.worker.id()}

    @hatchet.step()
    def step1b(self, context: Context) -> dict[str, str | None]:
        return {"worker": context.worker.id()}

    @hatchet.step(parents=["step1a", "step1b"])
    async def step2(self, context: Context) -> dict[str, str | None]:
        ref = await context.aio_spawn_workflow(
            "StickyChildWorkflow", {}, options=ChildTriggerWorkflowOptions(sticky=True)
        )

        await ref.result()

        return {"worker": context.worker.id()}


sticky_child_workflow = hatchet.declare_workflow(
    on_events=["sticky:child"], sticky=StickyStrategy.SOFT
)


class StickyChildWorkflow(BaseWorkflow):
    config = sticky_child_workflow.config

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
