import asyncio
import time
from datetime import timedelta
from typing import Any
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


dag_child_workflow = hatchet.workflow(name="dag-child-workflow")


@dag_child_workflow.task()
async def dag_child_1(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(1)
    return {"result": "child1"}


@dag_child_workflow.task()
async def dag_child_2(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(5)
    return {"result": "child2"}


@hatchet.durable_task()
async def durable_spawn_dag(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    sleep_start = time.time()
    sleep_result = await ctx.aio_sleep_for(timedelta(seconds=1))
    sleep_duration = time.time() - sleep_start

    spawn_start = time.time()
    spawn_result = await dag_child_workflow.aio_run()
    spawn_duration = time.time() - spawn_start

    return {
        "sleep_duration": sleep_duration,
        "sleep_result": sleep_result,
        "spawn_duration": spawn_duration,
        "spawn_result": spawn_result,
    }


# > Create a durable workflow
durable_workflow = hatchet.workflow(name="DurableWorkflow")


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


@durable_workflow.durable_task()
async def wait_for_multi_sleep(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | int]:
    start = time.time()

    for _ in range(3):
        await ctx.aio_sleep_for(
            timedelta(seconds=SLEEP_TIME),
        )

    return {
        "runtime": int(time.time() - start),
    }


@ephemeral_workflow.task()
def ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


@hatchet.durable_task()
async def wait_for_sleep_twice(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, int]:
    try:
        start = time.time()

        await ctx.aio_sleep_for(
            timedelta(seconds=SLEEP_TIME),
        )

        return {
            "runtime": int(time.time() - start),
        }
    except asyncio.CancelledError:
        return {"runtime": -1}


@hatchet.task()
def spawn_child_task(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"message": "hello from child"}


@hatchet.durable_task()
async def durable_with_spawn(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    child_result = await spawn_child_task.aio_run()
    return {"child_output": child_result}


@hatchet.durable_task()
async def durable_sleep_event_spawn(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    start = time.time()

    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_TIME))

    await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key=EVENT_KEY, expression="true"),
    )

    child_result = await spawn_child_task.aio_run()

    return {
        "runtime": int(time.time() - start),
        "child_output": child_result,
    }


def main() -> None:
    worker = hatchet.worker(
        "durable-worker",
        workflows=[
            durable_workflow,
            ephemeral_workflow,
            wait_for_sleep_twice,
            spawn_child_task,
            durable_with_spawn,
            durable_sleep_event_spawn,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
