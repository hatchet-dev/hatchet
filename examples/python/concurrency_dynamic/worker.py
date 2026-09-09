from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


class WorkflowInput(BaseModel):
    account: str = "default"
    tier: str = "free"


# > Dynamic Max Runs
# max_runs accepts an int or a CEL expression string. With an expression, each
# concurrency group's limit is computed from the task's input.
@hatchet.task(
    input_validator=WorkflowInput,
    concurrency=[
        ConcurrencyStrategy(
            expression="input.account",
            max_runs="input.tier == 'premium' ? 10 : 1",
            strategy="GROUP_ROUND_ROBIN",
        ),
    ],
)
def dynamic_task(input: WorkflowInput, ctx: Context) -> None:
    print("running for account", input.account)




def main() -> None:
    worker = hatchet.worker("concurrency-dynamic-worker", workflows=[dynamic_task])
    worker.start()


if __name__ == "__main__":
    main()
