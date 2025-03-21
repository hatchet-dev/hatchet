import random
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.waits import SleepCondition


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

dag_waiting_workflow = hatchet.workflow(name="DAGWaitingWorkflow")


@dag_waiting_workflow.task()
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(wait_for=[SleepCondition(timedelta(seconds=10))])
def step2(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_waiting_workflow.task(parents=[step1, step2])
def step3(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

    return RandomSum(sum=one + two)


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_waiting_workflow])

    worker.start()


if __name__ == "__main__":
    main()
