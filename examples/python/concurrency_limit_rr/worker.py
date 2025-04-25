import time

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet(debug=True)

# ❓ Concurrency Strategy With Key
class WorkflowInput(BaseModel):
    group: str

concurrency_limit_rr_workflow = hatchet.workflow(
    name="ConcurrencyDemoWorkflowRR",
    concurrency=ConcurrencyExpression(
        expression="input.group",
        max_runs=1,
        limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
    ),
    input_validator=WorkflowInput,
)
# ‼️

@concurrency_limit_rr_workflow.task()
def step1(input: WorkflowInput, ctx: Context) -> None:
    print("starting step1")
    time.sleep(2)
    print("finished step1")
    pass

def main() -> None:
    worker = hatchet.worker(
        "concurrency-demo-worker-rr",
        slots=10,
        workflows=[concurrency_limit_rr_workflow],
    )

    worker.start()

if __name__ == "__main__":
    main()
