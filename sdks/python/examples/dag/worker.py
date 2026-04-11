import random
import time
from datetime import timedelta
from typing import Annotated
from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet, Parent


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet()

# > Define a DAG
dag_workflow = hatchet.workflow(name="DAGWorkflow")
# !!


# > First task
@dag_workflow.task(execution_timeout=timedelta(seconds=5))
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!

# > Task with parents


@dag_workflow.task(execution_timeout=timedelta(seconds=5))
async def step2(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task()
async def step3(
    input: EmptyModel,
    ctx: Context,
    step_1: Annotated[StepOutput, Parent(step1)],
    step_2: Annotated[StepOutput, Parent(step2)],
) -> RandomSum:
    print(step_1)
    one = step_1.random_number
    two = step_2.random_number

    return RandomSum(sum=one + two)


# !!


@dag_workflow.task(parents=[step1, step3])
async def step4(input: EmptyModel, ctx: Context) -> dict[str, str]:
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


# > Declare a worker
def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])

    worker.start()


# !!

if __name__ == "__main__":
    main()
