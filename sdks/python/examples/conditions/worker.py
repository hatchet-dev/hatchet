# > Create a workflow

import random
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
    or_,
)

hatchet = Hatchet(debug=True)


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


task_condition_workflow = hatchet.workflow(name="TaskConditionWorkflow")

# !!


# > Add base task
@task_condition_workflow.task()
def start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!


# > Add wait for sleep
@task_condition_workflow.task(
    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]
)
def wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!


# > Add skip condition override
@task_condition_workflow.task(
    parents=[start, wait_for_sleep],
    skip_if=[ParentCondition(parent=start, expression="output.random_number > 0")],
)
def skip_with_multiple_parents(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!


# > Add skip on event
@task_condition_workflow.task(
    parents=[start],
    wait_for=[SleepCondition(timedelta(seconds=30))],
    skip_if=[UserEventCondition(event_key="skip_on_event:skip")],
)
def skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!


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


# !!


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


# !!


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


# !!


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[task_condition_workflow])

    worker.start()


if __name__ == "__main__":
    main()
