import random
import time
from datetime import timedelta

from pydantic import BaseModel

from hatchet_sdk import (
    Context,
    EmptyModel,
    Hatchet,
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
)


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

# > Define a DAG
dag_workflow = hatchet.workflow(
    name="DAGWorkflow",
    concurrency=[
        ConcurrencyExpression(
            expression="additional_metadata.abc",
            max_runs=2,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
        ),
        ConcurrencyExpression(
            expression="input.foobar",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
    ],
)
# !!


# > First task
@dag_workflow.task(execution_timeout=timedelta(seconds=5), concurrency=2)
def step1(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


# !!

# > Task with parents


@dag_workflow.task(
    execution_timeout=timedelta(seconds=5),
    concurrency=[
        ConcurrencyExpression(
            expression="additional_metadata.xyz",
            max_runs=3,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
        )
    ],
)
async def step2(input: EmptyModel, ctx: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(parents=[step1, step2])
async def step3(input: EmptyModel, ctx: Context) -> RandomSum:
    one = ctx.task_output(step1).random_number
    two = ctx.task_output(step2).random_number

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
