import random
import time

from pydantic import BaseModel

from hatchet_sdk import Context, EmptyModel, Hatchet


class StepOutput(BaseModel):
    random_number: int


class RandomSum(BaseModel):
    sum: int


hatchet = Hatchet(debug=True)

dag_workflow = hatchet.workflow(name="DAGWorkflow", schedule_timeout="10m")


@dag_workflow.task(timeout="5s")
def step1(input: EmptyModel, context: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(timeout="5s")
def step2(input: EmptyModel, context: Context) -> StepOutput:
    return StepOutput(random_number=random.randint(1, 100))


@dag_workflow.task(parents=[step1, step2])
def step3(input: EmptyModel, context: Context) -> RandomSum:
    one = context.task_output(step1).random_number
    two = context.task_output(step2).random_number

    return RandomSum(sum=one + two)


@dag_workflow.task(parents=[step1, step3])
def step4(input: EmptyModel, context: Context) -> dict[str, str]:
    print(
        "executed step4",
        time.strftime("%H:%M:%S", time.localtime()),
        input,
        context.task_output(step1),
        context.task_output(step3),
    )
    return {
        "step4": "step4",
    }


def main() -> None:
    worker = hatchet.worker("dag-worker", workflows=[dag_workflow])

    worker.start()


if __name__ == "__main__":
    main()
