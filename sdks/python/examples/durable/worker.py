from datetime import timedelta

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet, SleepCondition

hatchet = Hatchet(debug=True)

durable_workflow = hatchet.workflow(name="DurableWorkflow")
ephemeral_workflow = hatchet.workflow(name="EphemeralWorkflow")


@durable_workflow.task()
async def ephemeral_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


@durable_workflow.durable_task()
async def durable_task(input: EmptyModel, ctx: DurableContext) -> None:
    print("Waiting for signal")
    await ctx.wait_for("foobar", SleepCondition(duration=timedelta(seconds=10)))
    print("Signal received")


@ephemeral_workflow.task()
def ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


def main() -> None:
    worker = hatchet.worker("durable-worker", workflows=[durable_workflow])
    worker.start()


if __name__ == "__main__":
    main()
