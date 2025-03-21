import random
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    SleepCondition,
    UserEventCondition,
    or_,
)


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

dag_waiting_workflow = hatchet.workflow(name="DAGWaitingWorkflow")


@dag_waiting_workflow.task()
def start(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(
    parents=[start], wait_for=[SleepCondition(timedelta(seconds=10))]
)
def wait_for_sleep(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(
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


@dag_waiting_workflow.task(
    parents=[start],
    wait_for=[SleepCondition(timedelta(seconds=30))],
    skip_if=[UserEventCondition(event_key="skip_on_event:skip")],
)
def skip_on_event(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(
    parents=[start, wait_for_sleep, wait_for_event, skip_on_event],
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

    return RandomSum(sum=one + two + three + four)


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_waiting_workflow])

    worker.start()


if __name__ == "__main__":
    main()
