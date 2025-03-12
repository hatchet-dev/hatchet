from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    StickyStrategy,
    TriggerWorkflowOptions,
)

hatchet = Hatchet(debug=True)

sticky_workflow = hatchet.workflow(
    name="StickyWorkflow", on_events=["sticky:parent"], sticky=StickyStrategy.SOFT
)
sticky_child_workflow = hatchet.workflow(
    name="StickyChildWorkflow", on_events=["sticky:child"], sticky=StickyStrategy.SOFT
)


@sticky_workflow.task()
def step1a(input: EmptyModel, context: Context) -> dict[str, str | None]:
    return {"worker": context.worker.id()}


@sticky_workflow.task()
def step1b(input: EmptyModel, context: Context) -> dict[str, str | None]:
    return {"worker": context.worker.id()}


@sticky_workflow.task(parents=[step1a, step1b])
async def step2(input: EmptyModel, context: Context) -> dict[str, str | None]:
    ref = await sticky_child_workflow.aio_run(
        options=TriggerWorkflowOptions(sticky=True)
    )

    await ref.aio_result()

    return {"worker": context.worker.id()}


@sticky_child_workflow.task()
def child(input: EmptyModel, context: Context) -> dict[str, str | None]:
    return {"worker": context.worker.id()}


def main() -> None:
    worker = hatchet.worker(
        "sticky-worker", slots=10, workflows=[sticky_workflow, sticky_child_workflow]
    )
    worker.start()


if __name__ == "__main__":
    main()
