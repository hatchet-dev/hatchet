# > Create a workflow

import random
from datetime import timedelta

from pydantic import BaseModel

from typing import Any

from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
    or_,
)
from hatchet_sdk.runnables.workflow import BaseWorkflow

hatchet = Hatchet()


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


task_condition_workflow = hatchet.workflow(name="TaskConditionWorkflow")



# > Add base task
@task_condition_workflow.task()
def start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add wait for sleep
@task_condition_workflow.task(
    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]
)
def wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add skip condition override
@task_condition_workflow.task(
    parents=[start, wait_for_sleep],
    skip_if=[ParentCondition(parent=start, expression="output.random_number > 0")],
)
def skip_with_multiple_parents(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add skip on event
@task_condition_workflow.task(
    parents=[start],
    wait_for=[SleepCondition(timedelta(seconds=30))],
    skip_if=[UserEventCondition(event_key="skip_on_event:skip")],
)
def skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add branching
@task_condition_workflow.task(
    parents=[wait_for_sleep],
    skip_if=[
        ParentCondition(
            parent=wait_for_sleep,
            expression="output.random_number > 50",
        )
    ],
)
def left_branch(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@task_condition_workflow.task(
    parents=[wait_for_sleep],
    skip_if=[
        ParentCondition(
            parent=wait_for_sleep,
            expression="output.random_number <= 50",
        )
    ],
)
def right_branch(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add wait for event
@task_condition_workflow.task(
    parents=[start],
    wait_for=[
        or_(
            SleepCondition(duration=timedelta(minutes=1)),
            UserEventCondition(event_key="wait_for_event:start"),
        )
    ],
)
def wait_for_event(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add multiple or groups
@task_condition_workflow.task(
    parents=[start],
    wait_for=[
        or_(
            SleepCondition(duration=timedelta(seconds=30), readable_data_key="first"),
        ),
        or_(
            SleepCondition(duration=timedelta(seconds=30), readable_data_key="second"),
            UserEventCondition(event_key="payment:processed"),
        ),
    ],
)
def wait_for_or_groups(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))




# > Add sum
@task_condition_workflow.task(
    parents=[
        start,
        wait_for_sleep,
        wait_for_event,
        skip_on_event,
        left_branch,
        right_branch,
    ],
)
def sum(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(start).random_number
    two = ctx.task_output(wait_for_event).random_number
    three = ctx.task_output(wait_for_sleep).random_number
    four = (
        ctx.task_output(skip_on_event).random_number
        if not ctx.was_skipped(skip_on_event)
        else 0
    )

    five = (
        ctx.task_output(left_branch).random_number
        if not ctx.was_skipped(left_branch)
        else 0
    )
    six = (
        ctx.task_output(right_branch).random_number
        if not ctx.was_skipped(right_branch)
        else 0
    )

    return RandomSum(sum=one + two + three + four + five + six)




cancel_if_workflow = hatchet.workflow(name="CancelIfWorkflow")


@cancel_if_workflow.task()
async def start_cancel_if(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


@cancel_if_workflow.task(
    parents=[start_cancel_if],
    cancel_if=[
        ParentCondition(parent=start_cancel_if, expression="output.random_number > 1")
    ],
)
async def cancel_if(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=3)  # should not run


@cancel_if_workflow.task(
    parents=[cancel_if],
)
async def downstream_skip(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=3)  # should not run


skip_if_sleep_workflow = hatchet.workflow(name="SkipIfSleepWorkflow")


@skip_if_sleep_workflow.task()
def sis_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=1)


@skip_if_sleep_workflow.task(
    parents=[sis_start],
    wait_for=[UserEventCondition(event_key="skip_if_sleep:proceed")],
    skip_if=[SleepCondition(timedelta(seconds=8))],
)
def sis_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


skip_if_or_workflow = hatchet.workflow(name="SkipIfOrGroupWorkflow")


@skip_if_or_workflow.task()
def sio_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@skip_if_or_workflow.task(
    parents=[sio_start],
    skip_if=[
        or_(
            ParentCondition(parent=sio_start, expression="output.random_number >= 1"),
            ParentCondition(parent=sio_start, expression="output.random_number > 1000"),
        )
    ],
)
def sio_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


cancel_if_event_workflow = hatchet.workflow(name="CancelIfEventWorkflow")


@cancel_if_event_workflow.task()
def cie_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=1)


@cancel_if_event_workflow.task(
    parents=[cie_start],
    wait_for=[SleepCondition(timedelta(seconds=30))],
    cancel_if=[UserEventCondition(event_key="cancel_if_event:abort")],
)
def cie_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


@cancel_if_event_workflow.task(parents=[cie_target])
def cie_downstream(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=3)  # should not run


cancel_if_sleep_workflow = hatchet.workflow(name="CancelIfSleepWorkflow")


@cancel_if_sleep_workflow.task()
def cis_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=1)


@cancel_if_sleep_workflow.task(
    parents=[cis_start],
    wait_for=[UserEventCondition(event_key="cancel_if_sleep:proceed")],
    cancel_if=[SleepCondition(timedelta(seconds=8))],
)
def cis_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


@cancel_if_sleep_workflow.task(parents=[cis_target])
def cis_downstream(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=3)  # should not run


cancel_if_or_workflow = hatchet.workflow(name="CancelIfOrGroupWorkflow")


@cancel_if_or_workflow.task()
def cio_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=1)


@cancel_if_or_workflow.task(
    parents=[cio_start],
    wait_for=[UserEventCondition(event_key="cancel_if_or:proceed")],
    cancel_if=[
        or_(
            UserEventCondition(event_key="cancel_if_or:abort"),
            SleepCondition(timedelta(seconds=8)),
        )
    ],
)
def cio_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=2)


@cancel_if_or_workflow.task(parents=[cio_target])
def cio_downstream(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=3)  # should not run


wait_for_event_only_workflow = hatchet.workflow(name="WaitForEventOnlyWorkflow")


@wait_for_event_only_workflow.task()
def wfe_start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=1)


@wait_for_event_only_workflow.task(
    parents=[wfe_start],
    wait_for=[UserEventCondition(event_key="wait_for_event_only:go")],
)
def wfe_target(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=5)


condition_workflows: list[BaseWorkflow[Any]] = [
    task_condition_workflow,
    cancel_if_workflow,
    skip_if_sleep_workflow,
    skip_if_or_workflow,
    cancel_if_event_workflow,
    cancel_if_sleep_workflow,
    cancel_if_or_workflow,
    wait_for_event_only_workflow,
]


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=condition_workflows)

    worker.start()


if __name__ == "__main__":
    main()
