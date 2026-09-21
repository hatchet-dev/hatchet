import random
import time
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet


class DAGWorkflowInput(BaseModel):
    should_fail: bool = False


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet()

# > Define a DAG
dag_workflow = hatchet.workflow(name="DAGWorkflow", input_validator=DAGWorkflowInput)
# !!


# > First task
@dag_workflow.task(execution_timeout=timedelta(seconds=5))
def step1(input: DAGWorkflowInput, ctx: Context) -> StepOutput:
    if input.should_fail:
        raise Exception("intentional error")

    return StepOutput(random_number=random.randint(1, 100))


# !!

# > Task with parents


@dag_workflow.task(execution_timeout=timedelta(seconds=5))
async def step2(input: DAGWorkflowInput, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(parents=[step1, step2])
async def step3(input: DAGWorkflowInput, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

    return RandomSum(sum=one + two)


# !!


@dag_workflow.task(parents=[step1, step3])
async def step4(input: DAGWorkflowInput, ctx: Context) -> dict[str, str]:
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


@dag_workflow.on_failure_task()
def on_failure(input: DAGWorkflowInput, ctx: Context) -> dict[str, bool]:
    return {"ran": True}


# > Declare a worker
def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])

    worker.start()


# !!

if __name__ == "__main__":
    main()
