from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    StickyStrategy,
    TriggerWorkflowOptions,
)

hatchet = Hatchet(debug=True)

# > StickyWorker


sticky_workflow = hatchet.workflow(
    name="StickyWorkflow",
    # 👀 Specify a sticky strategy when declaring the workflow
    sticky=StickyStrategy.SOFT,
)


@sticky_workflow.task()
def step1a(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    return {"worker": ctx.worker.id()}


@sticky_workflow.task()
def step1b(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    return {"worker": ctx.worker.id()}


# !!

# > StickyChild

sticky_child_workflow = hatchet.workflow(
    name="StickyChildWorkflow", sticky=StickyStrategy.SOFT
)


@sticky_workflow.task(parents=[step1a, step1b])
async def step2(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    ref = await sticky_child_workflow.aio_run_no_wait(
        options=TriggerWorkflowOptions(sticky=True)
    )

    await ref.aio_result()

    return {"worker": ctx.worker.id()}


@sticky_child_workflow.task()
def child(input: EmptyModel, ctx: Context) -> dict[str, str | None]:
    return {"worker": ctx.worker.id()}


# !!


def main() -> None:
    worker = hatchet.worker(
        "sticky-worker", slots=10, workflows=[sticky_workflow, sticky_child_workflow]
    )
    worker.start()


if __name__ == "__main__":
    main()
