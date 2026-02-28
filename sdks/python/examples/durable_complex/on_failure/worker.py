from __future__ import annotations

from datetime import timedelta

from hatchet_sdk import Context, DurableContext, EmptyModel, Hatchet
from hatchet_sdk.exceptions import TaskRunError
from hatchet_sdk.runnables.eviction import EvictionPolicy

hatchet = Hatchet(debug=True)

ERROR_TEXT = "durable task failed"
SLEEP_SECONDS = 6
EVICTION_POLICY = EvictionPolicy(
    ttl=timedelta(seconds=1),
    allow_capacity_eviction=True,
    priority=0,
)

durable_on_failure_workflow = hatchet.workflow(name="DurableOnFailureWorkflow")


@durable_on_failure_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_failing_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    raise RuntimeError(ERROR_TEXT)


@durable_on_failure_workflow.on_failure_task()
def durable_on_failure_handler(input: EmptyModel, ctx: Context) -> dict[str, str | int]:
    errors = ctx.task_run_errors
    assert len(errors) > 0
    return {"status": "failure_handled", "error_count": len(errors)}


durable_on_success_workflow = hatchet.workflow(name="DurableOnSuccessWorkflow")


@durable_on_success_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_success_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    return {"status": "completed"}


@durable_on_success_workflow.on_success_task()
def durable_on_success_handler(input: EmptyModel, ctx: Context) -> dict[str, str]:
    return {"status": "success_handled"}


durable_on_failure_details_workflow = hatchet.workflow(
    name="DurableOnFailureDetailsWorkflow"
)


@durable_on_failure_details_workflow.durable_task(
    execution_timeout=timedelta(minutes=5),
    eviction_policy=EVICTION_POLICY,
)
async def durable_failing_details_task(
    input: EmptyModel, ctx: DurableContext
) -> dict[str, str]:
    await ctx.aio_sleep_for(timedelta(seconds=SLEEP_SECONDS))
    raise RuntimeError(ERROR_TEXT)


@durable_on_failure_details_workflow.on_failure_task()
def durable_on_failure_details_handler(
    input: EmptyModel, ctx: Context
) -> dict[str, str | None]:
    error = ctx.get_task_run_error(durable_failing_details_task)
    assert error is not None
    assert isinstance(error, TaskRunError)
    assert ERROR_TEXT in error.exc
    return {
        "status": "details_handled",
        "task_run_external_id": error.task_run_external_id,
    }


def main() -> None:
    worker = hatchet.worker(
        "durable-complex-on-failure-worker",
        workflows=[
            durable_on_failure_workflow,
            durable_on_success_workflow,
            durable_on_failure_details_workflow,
        ],
    )
    worker.start()


if __name__ == "__main__":
    main()
