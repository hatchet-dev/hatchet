import random
import time
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

dag_workflow = hatchet.workflow(
    name="DAGWorkflow", schedule_timeout=timedelta(minutes=10)
)


@dag_workflow.task(timeout=timedelta(seconds=5))
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(timeout=timedelta(seconds=5))
def step2(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(parents=[step1, step2])
def step3(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

    return RandomSum(sum=one + two)


@dag_workflow.task(parents=[step1, step3])
def step4(input: EmptyModel, ctx: Context) -> dict[str, str]:
    print(
        "executed step4",
        time.strftime("%H:%M:%S", time.localtime()),
        input,
        ctx.task_output(step1),
        ctx.task_output(step3),
    )
    return {
        "step4": "step4",
    }


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])

    worker.start()


if __name__ == "__main__":
    main()
