from typing import Annotated, Generator, AsyncGenerator

from pydantic import BaseModel

from hatchet_sdk import Context, Depends, DurableContext, EmptyModel, Hatchet
from contextlib import asynccontextmanager, contextmanager

hatchet = Hatchet(debug=False)

SYNC_DEPENDENCY_VALUE = "sync_dependency_value"
ASYNC_DEPENDENCY_VALUE = "async_dependency_value"
SYNC_CM_DEPENDENCY_VALUE = "sync_cm_dependency_value"
ASYNC_CM_DEPENDENCY_VALUE = "async_cm_dependency_value"


# > Declare dependencies
async def async_dep(input: EmptyModel, ctx: Context) -> str:
    return ASYNC_DEPENDENCY_VALUE


def sync_dep(input: EmptyModel, ctx: Context) -> str:
    return SYNC_DEPENDENCY_VALUE


@asynccontextmanager
async def async_cm_dep(input: EmptyModel, ctx: Context) -> AsyncGenerator[str, None]:
    value = ASYNC_CM_DEPENDENCY_VALUE
    try:
        yield value
    finally:
        pass


@contextmanager
def sync_cm_dep(input: EmptyModel, ctx: Context) -> Generator[str, None, None]:
    value = SYNC_CM_DEPENDENCY_VALUE
    try:
        yield value
    finally:
        pass


# !!


class Output(BaseModel):
    sync_dep: str
    async_dep: str
    async_cm_dep: str
    sync_cm_dep: str


# > Inject dependencies
@hatchet.task()
async def async_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


# !!


@hatchet.task()
def sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


@hatchet.durable_task()
async def durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


@hatchet.durable_task()
def durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
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
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


@di_workflow.task()
def wf_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


@di_workflow.durable_task()
async def wf_durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
    )


@di_workflow.durable_task()
def wf_durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
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
