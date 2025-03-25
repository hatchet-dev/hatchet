from datetime import timedelta

from hatchet_sdk import (
    Context,
    DurableContext,
    EmptyModel,
    Hatchet,
    SleepCondition,
    UserEventCondition,
)

hatchet = Hatchet(debug=True)

# ❓ Create a durable workflow
durable_workflow = hatchet.workflow(name="DurableWorkflow")
# !!


ephemeral_workflow = hatchet.workflow(name="EphemeralWorkflow")



# ❓ Add durable task
EVENT_KEY = "durable-example:event"
SLEEP_TIME = 5


@durable_workflow.task()
async def ephemeral_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


@durable_workflow.durable_task()
async def durable_task(input: EmptyModel, ctx: DurableContext) -> None:
    print("Waiting for sleep")
    await ctx.wait_for("sleep", SleepCondition(duration=timedelta(seconds=SLEEP_TIME)))
    print("Sleep finished")

    print("Waiting for event")
    await ctx.wait_for(
        "event",
        UserEventCondition(event_key=EVENT_KEY, expression="true"),
    )
    print("Event received")

# !!

@ephemeral_workflow.task()
def ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")




def main() -> None:
    worker = hatchet.worker(
        "durable-worker", workflows=[durable_workflow, ephemeral_workflow]
    )
    worker.start()


if __name__ == "__main__":
    main()
