import asyncio
import time
from datetime import timedelta
from uuid import uuid4

from hatchet_sdk import (
    Context,
    DurableContext,
    EmptyModel,
    Hatchet,
    UserEventCondition,
)
from hatchet_sdk.runnables.eviction import EvictionPolicy
from hatchet_sdk.worker.durable_eviction.manager import DurableEvictionConfig

hatchet = Hatchet(debug=True)

# > Add durable task
EVENT_KEY = "durable-eviction-example:event"
EVICTION_TIME = timedelta(seconds=3)
SLEEP_TIME = 10


@hatchet.task()
async def ephemeral_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")
    await asyncio.sleep(SLEEP_TIME)

@hatchet.task()
async def emit_event_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task 2")
    await asyncio.sleep(SLEEP_TIME)
    await hatchet.event.aio_push(EVENT_KEY, {})
    print("Event emitted")
    await asyncio.sleep(SLEEP_TIME)


@hatchet.durable_task(
    eviction=EvictionPolicy(
        ttl=timedelta(seconds=5),
    ),
)
async def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    print("Waiting for non-durable task")
    await ephemeral_task.aio_run()
    print("Non-durable task finished")

    print("Waiting for sleep")
    await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))
    print("Sleep finished")

    print("Pushing event")
    await emit_event_task.aio_run_no_wait()
    print("Waiting for event")
    await ctx.aio_wait_for(
        "event",
        UserEventCondition(event_key=EVENT_KEY, expression="true"),
    )
    print("Event received")

    return {
        "status": "success",
    }


def main() -> None:
    worker = hatchet.worker(
        "durable-worker",
        workflows=[durable_task, ephemeral_task, emit_event_task],
        durable_eviction_config=DurableEvictionConfig(
            check_interval=timedelta(seconds=1),
            reserve_slots=1,
            min_wait_for_capacity_eviction=timedelta(seconds=0),
        ),
    )
    worker.start()


if __name__ == "__main__":
    main()
