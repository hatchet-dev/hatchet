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
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(
    parents=[step1], wait_for=[SleepCondition(timedelta(seconds=10))]
)
def step2(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(
    parents=[step1],
    wait_for=[
        or_(
            SleepCondition(timedelta(seconds=15)),
            UserEventCondition(event_key="step3:start"),
        )
    ],
    skip_if=[UserEventCondition(event_key="step3:skip")],
)
def step3(input: EmptyModel, ctx: Context) -> RandomSum:
    raise Exception("This task should be skipped")


@dag_waiting_workflow.task(
    parents=[step2],
)
def step4(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

    return RandomSum(sum=one + two)


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_waiting_workflow])

    worker.start()


if __name__ == "__main__":
    main()
