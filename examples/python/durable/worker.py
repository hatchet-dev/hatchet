import asyncio
import time
from datetime import timedelta
from typing import Any
from uuid import uuid4

from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    DurableContext,
    EmptyModel,
    Hatchet,
    SleepCondition,
    UserEventCondition,
    or_,
)
from hatchet_sdk.exceptions import NonDeterminismError

hatchet = Hatchet()


dag_child_workflow = hatchet.workflow(name="dag-child-workflow")


@dag_child_workflow.task()
async def dag_child_1(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(1)
    return {"result": "child1"}


@dag_child_workflow.task(parents=[dag_child_1])
async def dag_child_2(input: EmptyModel, ctx: Context) -> dict[str, str]:
    await asyncio.sleep(5)
    return {"result": "child2"}


@hatchet.durable_task(execution_timeout=timedelta(seconds=10))
async def durable_spawn_dag(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    # NOTE: typically its not safe to use time.time() in a durable task, but
    # this test assumes that the task is not replayed or evicted and it is
    # used to ensure that the waits are accurate relative to the single invocation.
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
REPLAY_RESET_SLEEP_TIME = 3


@durable_workflow.task()
async def ephemeral_task(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


class AwaitedEvent(BaseModel):
    id: str


@durable_workflow.durable_task()
async def durable_task(input: EmptyModel, ctx: DurableContext) -> dict[str, str | int]:
    print("Waiting for sleep")
    sleep = await ctx.aio_sleep_for(duration=timedelta(seconds=SLEEP_TIME))
    print("Sleep finished")

    print("Waiting for event")
    event = await ctx.aio_wait_for_event(
        EVENT_KEY, "true", payload_validator=AwaitedEvent
    )
    print("Event received")

    return {
        "status": "success",
        "event_id": event.id,
        "sleep_duration_seconds": sleep.duration.seconds,
    }




# > Add durable tasks that wait for or groups


@durable_workflow.durable_task()
async def wait_for_or_group_1(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | int | float]:
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
        "runtime": time.time() - start,
        "key": key,
        "event_id": event_id,
    }




@durable_workflow.durable_task()
async def wait_for_or_group_2(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | int | float]:
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
        "runtime": time.time() - start,
        "key": key,
        "event_id": event_id,
    }


@durable_workflow.durable_task()
async def wait_for_multi_sleep(
    _i: EmptyModel, ctx: DurableContext
) -> dict[str, str | float]:
    start = time.time()

    for _ in range(3):
        await ctx.aio_sleep_for(
            timedelta(seconds=SLEEP_TIME),
        )

    return {
        "runtime": time.time() - start,
    }


@ephemeral_workflow.task()
def ephemeral_task_2(input: EmptyModel, ctx: Context) -> None:
    print("Running non-durable task")


@hatchet.durable_task()
async def memo_now_caching(_i: EmptyModel, ctx: DurableContext) -> dict[str, str]:
    now = await ctx.aio_now()
    return {
        "start_time": now.isoformat(),
    }


@hatchet.durable_task()
async def wait_for_sleep_twice(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, float]:
    try:
        start = time.time()

        await ctx.aio_sleep_for(
            timedelta(seconds=SLEEP_TIME),
        )

        return {
            "runtime": time.time() - start,
        }
    except asyncio.CancelledError:
        return {"runtime": -1.0}


class DurableBulkSpawnInput(BaseModel):
    n: int = 1


@hatchet.task(input_validator=DurableBulkSpawnInput)
def spawn_child_task(input: DurableBulkSpawnInput, ctx: Context) -> dict[str, str]:
    return {"message": "hello from child " + str(input.n)}


@hatchet.durable_task(execution_timeout=timedelta(seconds=10))
async def durable_with_spawn(input: EmptyModel, ctx: DurableContext) -> dict[str, Any]:
    child_result = await spawn_child_task.aio_run()
    return {"child_output": child_result}


@hatchet.durable_task(input_validator=DurableBulkSpawnInput)
async def durable_with_bulk_spawn(
    input: DurableBulkSpawnInput, ctx: DurableContext
) -> dict[str, Any]:
    child_results = await spawn_child_task.aio_run_many(
        [
            spawn_child_task.create_bulk_run_item(
                input=DurableBulkSpawnInput(n=i),
            )
            for i in range(input.n)
        ]
    )
    return {"child_outputs": child_results}


@hatchet.durable_task()
async def durable_sleep_event_spawn(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, Any]:
    start = time.time()

    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_TIME))

    await ctx.aio_wait_for_event(
        EVENT_KEY,
        "true",
    )

    child_result = await spawn_child_task.aio_run()

    return {
        "runtime": time.time() - start,
        "child_output": child_result,
    }


class EventLookbackInput(BaseModel):
    scope: str


class LookbackEventPayload(BaseModel):
    order: str


class EventLookbackResult(BaseModel):
    elapsed: float


class EventLookbackResultWithEvent(EventLookbackResult):
    event: LookbackEventPayload


class TwoEventsResult(BaseModel):
    event1: LookbackEventPayload
    event2: LookbackEventPayload
    elapsed: float


@hatchet.durable_task(input_validator=EventLookbackInput)
async def wait_for_event_lookback(
    input: EventLookbackInput, ctx: DurableContext
) -> EventLookbackResultWithEvent:
    start = time.time()
    event = await ctx.aio_wait_for_event(
        EVENT_KEY,
        scope=input.scope,
        lookback_window=timedelta(minutes=1),
        payload_validator=LookbackEventPayload,
    )
    return EventLookbackResultWithEvent(event=event, elapsed=time.time() - start)


@hatchet.durable_task(input_validator=EventLookbackInput)
async def wait_for_or_event_lookback(
    input: EventLookbackInput, ctx: DurableContext
) -> EventLookbackResult:
    start = time.time()
    now = await ctx.aio_now()
    consider_events_since = now - timedelta(minutes=1)

    await ctx.aio_wait_for(
        "or-event-lookback",
        or_(
            SleepCondition(timedelta(seconds=SLEEP_TIME)),
            UserEventCondition(
                event_key=EVENT_KEY,
                scope=input.scope,
                consider_events_since=consider_events_since,
            ),
        ),
    )

    return EventLookbackResult(elapsed=time.time() - start)


@hatchet.durable_task(input_validator=EventLookbackInput)
async def wait_for_two_events_second_pushed_first(
    input: EventLookbackInput, ctx: DurableContext
) -> TwoEventsResult:
    start = time.time()
    event1 = await ctx.aio_wait_for_event(
        "key1",
        scope=input.scope,
        lookback_window=timedelta(minutes=1),
        payload_validator=LookbackEventPayload,
    )
    event2 = await ctx.aio_wait_for_event(
        "key2",
        scope=input.scope,
        lookback_window=timedelta(minutes=1),
        payload_validator=LookbackEventPayload,
    )
    return TwoEventsResult(event1=event1, event2=event2, elapsed=time.time() - start)


class NonDeterminismOutput(BaseModel):
    attempt_number: int
    sleep_time: int

    non_determinism_detected: bool = False
    node_id: int | None = None


@hatchet.durable_task(execution_timeout=timedelta(seconds=10))
async def durable_non_determinism(
    input: EmptyModel, ctx: DurableContext
) -> NonDeterminismOutput:
    sleep_time = ctx.attempt_number * 2

    try:
        await ctx.aio_sleep_for(timedelta(seconds=sleep_time))
    except NonDeterminismError as e:
        return NonDeterminismOutput(
            attempt_number=ctx.attempt_number,
            sleep_time=sleep_time,
            non_determinism_detected=True,
            node_id=e.node_id,
        )

    return NonDeterminismOutput(
        attempt_number=ctx.attempt_number,
        sleep_time=sleep_time,
    )


class ReplayResetResponse(BaseModel):
    sleep_1_duration: float
    sleep_2_duration: float
    sleep_3_duration: float


@hatchet.durable_task(execution_timeout=timedelta(seconds=20))
async def durable_replay_reset(
    input: EmptyModel, ctx: DurableContext
) -> ReplayResetResponse:
    start = time.time()
    await ctx.aio_sleep_for(timedelta(seconds=REPLAY_RESET_SLEEP_TIME))
    sleep_1_duration = time.time() - start

    start = time.time()
    await ctx.aio_sleep_for(timedelta(seconds=REPLAY_RESET_SLEEP_TIME))
    sleep_2_duration = time.time() - start

    start = time.time()
    await ctx.aio_sleep_for(timedelta(seconds=REPLAY_RESET_SLEEP_TIME))
    sleep_3_duration = time.time() - start

    return ReplayResetResponse(
        sleep_1_duration=sleep_1_duration,
        sleep_2_duration=sleep_2_duration,
        sleep_3_duration=sleep_3_duration,
    )


class SleepResult(BaseModel):
    message: str
    duration: float


class MemoInput(BaseModel):
    message: str


async def expensive_computation(message: str) -> SleepResult:
    await asyncio.sleep(SLEEP_TIME)

    return SleepResult(message=message, duration=SLEEP_TIME)


@hatchet.durable_task(input_validator=MemoInput)
async def memo_task(input: MemoInput, ctx: DurableContext) -> SleepResult:
    start = time.time()
    res = await ctx._aio_memo(
        expensive_computation,
        SleepResult,
        input.message,
    )

    return SleepResult(message=res.message, duration=time.time() - start)


def main() -> None:
    worker = hatchet.worker(
        "durable-worker",
        workflows=[
            durable_workflow,
            ephemeral_workflow,
            wait_for_sleep_twice,
            spawn_child_task,
            durable_with_spawn,
            durable_with_bulk_spawn,
            durable_sleep_event_spawn,
            durable_non_determinism,
            durable_replay_reset,
            wait_for_event_lookback,
            wait_for_or_event_lookback,
            wait_for_two_events_second_pushed_first,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
