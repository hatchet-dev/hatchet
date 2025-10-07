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

# > Define a DAG
dag_workflow = hatchet.workflow(name="DAGWorkflow")
# !!


# > First task
@dag_workflow.task(execution_timeout=timedelta(seconds=5))
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!

# > Task with parents


@dag_workflow.task(parents=[step1])
async def step2(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number

    return RandomSum(sum=one)


# !!


# > Declare a worker
def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])

    worker.start()


# !!

if __name__ == "__main__":
    main()
