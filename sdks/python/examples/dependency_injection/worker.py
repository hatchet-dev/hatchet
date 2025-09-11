from typing import Annotated

from pydantic import BaseModel

from hatchet_sdk import Context, Depends, DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=False)

SYNC_DEPENDENCY_VALUE = "sync_dependency_value"
ASYNC_DEPENDENCY_VALUE = "async_dependency_value"


# > Declare dependencies
async def async_dep(input: EmptyModel, ctx: Context) -> str:
    return ASYNC_DEPENDENCY_VALUE


def sync_dep(input: EmptyModel, ctx: Context) -> str:
    return SYNC_DEPENDENCY_VALUE


# !!


class Output(BaseModel):
    sync_dep: str
    async_dep: str


# > Inject dependencies
@hatchet.task()
async def async_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


# !!


@hatchet.task()
def sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


@hatchet.durable_task()
async def durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


@hatchet.durable_task()
def durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


di_workflow = hatchet.workflow(
    name="dependency-injection-workflow",
)


@di_workflow.task()
async def wf_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


@di_workflow.task()
def wf_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


@di_workflow.durable_task()
async def wf_durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


@di_workflow.durable_task()
def wf_durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )


def main() -> None:
    worker = hatchet.worker(
        "dependency-injection-worker",
        workflows=[
            async_task_with_dependencies,
            sync_task_with_dependencies,
            durable_async_task_with_dependencies,
            durable_sync_task_with_dependencies,
            di_workflow,
        ],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
