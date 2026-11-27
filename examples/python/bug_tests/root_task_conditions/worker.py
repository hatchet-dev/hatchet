from datetime import timedelta

from hatchet_sdk import Context, Hatchet, SleepCondition, UserEventCondition

hatchet = Hatchet()

ROOT_WAIT_FOR_EVENT_KEY = "root-task-conditions:wait-for-event"
ROOT_SKIP_IF_EVENT_KEY = "root-task-conditions:skip-if-event"
ROOT_CANCEL_IF_SLEEP_NEVER_PUSHED_EVENT_KEY = (
    "root-task-conditions:cancel-if-sleep-never-pushed"
)

root_wait_for_sleep = hatchet.workflow(name="root-task-wait-for-sleep")


@root_wait_for_sleep.task(wait_for=[SleepCondition(timedelta(seconds=6))])
async def root_wait_for_sleep_task(input: None, ctx: Context) -> dict[str, bool]:
    return {"ran": True}


root_wait_for_event = hatchet.workflow(name="root-task-wait-for-event")


@root_wait_for_event.task(
    wait_for=[UserEventCondition(event_key=ROOT_WAIT_FOR_EVENT_KEY)]
)
async def root_wait_for_event_task(input: None, ctx: Context) -> dict[str, bool]:
    return {"ran": True}


root_skip_if_event = hatchet.workflow(name="root-task-skip-if-event")


@root_skip_if_event.task(
    wait_for=[SleepCondition(timedelta(seconds=60))],
    skip_if=[UserEventCondition(event_key=ROOT_SKIP_IF_EVENT_KEY)],
)
async def root_skip_if_event_task(input: None, ctx: Context) -> dict[str, bool]:
    return {"ran": True}


root_cancel_if_sleep = hatchet.workflow(name="root-task-cancel-if-sleep")


@root_cancel_if_sleep.task(
    wait_for=[
        UserEventCondition(event_key=ROOT_CANCEL_IF_SLEEP_NEVER_PUSHED_EVENT_KEY)
    ],
    cancel_if=[SleepCondition(timedelta(seconds=3))],
)
async def root_cancel_if_sleep_task(input: None, ctx: Context) -> dict[str, bool]:
    return {"ran": True}


root_task_condition_workflows = [
    root_wait_for_sleep,
    root_wait_for_event,
    root_skip_if_event,
    root_cancel_if_sleep,
]
