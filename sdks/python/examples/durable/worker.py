import time
from datetime import timedelta
from uuid import uuid4

from hatchet_sdk import (
    Context,
    DurableContext,
    EmptyModel,
    Hatchet,
    SleepCondition,
    UserEventCondition,
    or_,
)

hatchet = Hatchet(debug=True)

# > Create a durable workflow
durable_workflow = hatchet.workflow(name="DurableWorkflow")
# !!


ephemeral_workflow = hatchet.workflow(name="EphemeralWorkflow")


# > Add durable task
EVENT_KEY = "durable-example:event"
SLEEP_TIME = 5


@durable_workflow.task()
async def ephemeral_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


@durable_workflow.durable_task()
async def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    print("Waiting for sleep")
    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))
    print("Sleep finished")

    print("Waiting for event")
    await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key=EVENT_KEY, expression="true"),
    )
    print("Event received")

    return {
        "status": "success",
    }


# !!


# > Add durable tasks that wait for or groups


@durable_workflow.durable_task()
async def wait_for_or_group_1(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    start = time.time()
    wait_result = await ctx.aio_wait_for(
        uuid4().hex,
        or_(
            SleepCondition(timedelta(seconds=SLEEP_TIME)),
            UserEventCondition(event_key=EVENT_KEY),
        ),
    )

    key = list(wait_result.keys())[0]
    event_id = list(wait_result[key].keys())[0]

    return {
        "runtime": int(time.time() - start),
        "key": key,
        "event_id": event_id,
    }


@durable_workflow.durable_task()
async def wait_for_or_group_2(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    start = time.time()
    wait_result = await ctx.aio_wait_for(
        uuid4().hex,
        or_(
            SleepCondition(timedelta(seconds=6 * SLEEP_TIME)),
            UserEventCondition(event_key=EVENT_KEY),
        ),
    )

    key = list(wait_result.keys())[0]
    event_id = list(wait_result[key].keys())[0]

    return {
        "runtime": int(time.time() - start),
        "key": key,
        "event_id": event_id,
    }


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
