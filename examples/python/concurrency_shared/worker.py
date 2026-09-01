import time

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyExpression,
    ConcurrencyLimitStrategy,
    Context,
    Hatchet,
    SharedConcurrency,
)

hatchet = Hatchet()


class WorkflowInput(BaseModel):
    group: str = "default"
    inline: str = "default"
    chain_c: str = "default"


# > Shared Concurrency Strategy
# A tenant-scoped strategy is shared across workflows: every task declaring the same name
# consumes the same concurrency limit. The definition rides on workflow registration and
# re-registering the name updates it in place.
shared_limit = SharedConcurrency(
    name="example-shared-limit",
    expression="input.group",
    max_runs=1,
    limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
)

concurrency_shared_workflow_a = hatchet.workflow(
    name="ConcurrencySharedA",
    input_validator=WorkflowInput,
)

concurrency_shared_workflow_b = hatchet.workflow(
    name="ConcurrencySharedB",
    input_validator=WorkflowInput,
)


@concurrency_shared_workflow_a.task(concurrency=[shared_limit])
def task_a(input: WorkflowInput, ctx: Context) -> dict[str, int]:
    start = int(time.time() * 1000)
    time.sleep(1.5)
    return {"start_ms": start, "end_ms": int(time.time() * 1000)}


@concurrency_shared_workflow_b.task(concurrency=[shared_limit])
def task_b(input: WorkflowInput, ctx: Context) -> dict[str, int]:
    start = int(time.time() * 1000)
    time.sleep(1.5)
    return {"start_ms": start, "end_ms": int(time.time() * 1000)}



# > Mixed Inline And Shared Concurrency
# A single task can combine a workflow-scoped inline strategy with a shared strategy;
# both limits apply at once.
concurrency_shared_mixed_workflow = hatchet.workflow(
    name="ConcurrencySharedMixed",
    input_validator=WorkflowInput,
)


@concurrency_shared_mixed_workflow.task(
    concurrency=[
        ConcurrencyExpression(
            expression="input.inline",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
        ),
        shared_limit,
    ],
)
def task_mixed(input: WorkflowInput, ctx: Context) -> dict[str, int]:
    start = int(time.time() * 1000)
    time.sleep(1.5)
    return {"start_ms": start, "end_ms": int(time.time() * 1000)}



# > Multi-Strategy Chain
# A chain can mix multiple tenant-scoped and workflow-scoped entries, each with its own
# limit strategy; entries are processed in the declared order.
chain_limit_a = SharedConcurrency(
    name="example-chain-limit-a",
    expression="input.group",
    max_runs=1,
    limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
)

chain_limit_c = SharedConcurrency(
    name="example-chain-limit-c",
    expression="input.chain_c",
    max_runs=1,
    limit_strategy=ConcurrencyLimitStrategy.GROUP_ROUND_ROBIN,
)

concurrency_shared_chain_workflow = hatchet.workflow(
    name="ConcurrencySharedChain",
    input_validator=WorkflowInput,
)


@concurrency_shared_chain_workflow.task(
    concurrency=[
        chain_limit_a,
        ConcurrencyExpression(
            expression="input.inline",
            max_runs=1,
            limit_strategy=ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
        ),
        chain_limit_c,
    ],
)
def task_chain(input: WorkflowInput, ctx: Context) -> dict[str, int]:
    start = int(time.time() * 1000)
    time.sleep(10)
    return {"start_ms": start, "end_ms": int(time.time() * 1000)}




def main() -> None:
    worker = hatchet.worker(
        "concurrency-shared-worker",
        workflows=[
            concurrency_shared_workflow_a,
            concurrency_shared_workflow_b,
            concurrency_shared_mixed_workflow,
            concurrency_shared_chain_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
