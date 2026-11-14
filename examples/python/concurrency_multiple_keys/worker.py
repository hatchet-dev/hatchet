import asyncio

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()

SLEEP_TIME = 2
DIGIT_MAX_RUNS = 8
NAME_MAX_RUNS = 3


# > Concurrency Strategy With Key
class WorkflowInput(BaseModel):
    name: str
    digit: str


concurrency_multiple_keys_workflow = hatchet.workflow(
    name="ConcurrencyWorkflowManyKeys",
    input_validator=WorkflowInput,
)


@concurrency_multiple_keys_workflow.task(
    concurrency=[
        ConcurrencyStrategy(
            expression="input.digit",
            max_runs=DIGIT_MAX_RUNS,
            strategy="GROUP_ROUND_ROBIN",
        ),
        ConcurrencyStrategy(
            expression="input.name",
            max_runs=NAME_MAX_RUNS,
            strategy="GROUP_ROUND_ROBIN",
        ),
    ]
)
async def concurrency_task(input: WorkflowInput, ctx: Context) -> None:
    await asyncio.sleep(SLEEP_TIME)


def main() -> None:
    worker = hatchet.worker(
        "concurrency-worker-multiple-keys",
        slots=10,
        workflows=[concurrency_multiple_keys_workflow],
    )

    worker.start()


if __name__ == "__main__":
    main()
