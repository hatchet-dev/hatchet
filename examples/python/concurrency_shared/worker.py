import time

from pydantic import BaseModel

from hatchet_sdk import (
    ConcurrencyStrategy,
    Context,
    Hatchet,
)

hatchet = Hatchet()


class WorkflowInput(BaseModel):
    group: str = "default"
    inline: str = "default"
    chain_c: str = "default"


class RunWindow(BaseModel):
    """When the task function was actually executing, so tests can assert whether two
    runs overlapped."""

    start_ms: int
    end_ms: int


def run_window(duration_seconds: float) -> RunWindow:
    start = int(time.time() * 1000)
    time.sleep(duration_seconds)
    return RunWindow(start_ms=start, end_ms=int(time.time() * 1000))


# > Shared Concurrency Strategy
# A tenant-scoped strategy is shared across workflows: every task declaring the same name
# consumes the same concurrency limit. The definition rides on workflow registration and
# re-registering the name updates it in place.
shared_limit = ConcurrencyStrategy(
    expression="input.group",
    max_runs=1,
    strategy="GROUP_ROUND_ROBIN",
    name="example-shared-limit",
    is_tenant_scoped=True,
)


@hatchet.task(input_validator=WorkflowInput, concurrency=[shared_limit])
def task_a(input: WorkflowInput, ctx: Context) -> RunWindow:
    return run_window(1.5)


@hatchet.task(input_validator=WorkflowInput, concurrency=[shared_limit])
def task_b(input: WorkflowInput, ctx: Context) -> RunWindow:
    return run_window(1.5)




# > Mixed Inline And Shared Concurrency
# A single task can combine a workflow-scoped inline strategy with a shared strategy;
# both limits apply at once.
@hatchet.task(
    input_validator=WorkflowInput,
    concurrency=[
        ConcurrencyStrategy(
            expression="input.inline",
            max_runs=1,
            strategy="GROUP_ROUND_ROBIN",
        ),
        shared_limit,
    ],
)
def task_mixed(input: WorkflowInput, ctx: Context) -> RunWindow:
    return run_window(1.5)



# > Multi-Strategy Chain
# A chain can mix multiple tenant-scoped and workflow-scoped entries, each with its own
# limit strategy; entries are processed in the declared order.
chain_limit_a = ConcurrencyStrategy(
    expression="input.group",
    max_runs=1,
    strategy="GROUP_ROUND_ROBIN",
    name="example-chain-limit-a",
    is_tenant_scoped=True,
)

chain_limit_c = ConcurrencyStrategy(
    expression="input.chain_c",
    max_runs=1,
    strategy="GROUP_ROUND_ROBIN",
    name="example-chain-limit-c",
    is_tenant_scoped=True,
)


@hatchet.task(
    input_validator=WorkflowInput,
    concurrency=[
        chain_limit_a,
        ConcurrencyStrategy(
            expression="input.inline",
            max_runs=1,
            strategy="CANCEL_IN_PROGRESS",
        ),
        chain_limit_c,
    ],
)
def task_chain(input: WorkflowInput, ctx: Context) -> RunWindow:
    return run_window(10)




def main() -> None:
    worker = hatchet.worker(
        "concurrency-shared-worker",
        workflows=[
            task_a,
            task_b,
            task_mixed,
            task_chain,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
