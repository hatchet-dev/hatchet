from contextlib import asynccontextmanager, contextmanager
from typing import Annotated, AsyncGenerator, Generator
import sys

from pydantic import BaseModel

from hatchet_sdk import Context, Depends, DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=False)

SYNC_DEPENDENCY_VALUE = "sync_dependency_value"
ASYNC_DEPENDENCY_VALUE = "async_dependency_value"
SYNC_CM_DEPENDENCY_VALUE = "sync_cm_dependency_value"
ASYNC_CM_DEPENDENCY_VALUE = "async_cm_dependency_value"
CHAINED_CM_VALUE = "chained_cm_value"
CHAINED_ASYNC_CM_VALUE = "chained_async_cm_value"

if sys.version_info >= (3, 12):
    from examples.dependency_injection.dependency_annotations312 import (
        AsyncDepNoTypeAlias,
        AsyncDepTypeAlias,
        SyncDepNoTypeAlias,
        AsyncDepTypeSyntax,
        SyncDepTypeAlias,
        SyncDepTypeSyntax,
    )
else:
    from examples.dependency_injection.dependency_annotations310 import (
        AsyncDepNoTypeAlias,
        AsyncDepTypeAlias,
        SyncDepNoTypeAlias,
        AsyncDepTypeSyntax,
        SyncDepTypeAlias,
        SyncDepTypeSyntax,
    )


@hatchet.task()
async def task_with_type_aliases(
    _i: EmptyModel,
    ctx: Context,
    async_dep_no_type_alias: AsyncDepNoTypeAlias,
    async_dep_type_alias: AsyncDepTypeAlias,
    async_dep_type_syntax: AsyncDepTypeSyntax,
    sync_dep_no_type_alias: SyncDepNoTypeAlias,
    sync_dep_type_alias: SyncDepTypeAlias,
    sync_dep_type_syntax: SyncDepTypeSyntax,
) -> dict[str, bool]:
    return {
        "async_dep_no_type_alias": async_dep_no_type_alias,
        "async_dep_type_alias": async_dep_type_alias,
        "async_dep_type_syntax": async_dep_type_syntax,
        "sync_dep_no_type_alias": sync_dep_no_type_alias,
        "sync_dep_type_alias": sync_dep_type_alias,
        "sync_dep_type_syntax": sync_dep_type_syntax,
    }


# > Declare dependencies
async def async_dep(input: EmptyModel, ctx: Context) -> str:
    return ASYNC_DEPENDENCY_VALUE


def sync_dep(input: EmptyModel, ctx: Context) -> str:
    return SYNC_DEPENDENCY_VALUE


@asynccontextmanager
async def async_cm_dep(
    input: EmptyModel, ctx: Context, async_dep: Annotated[str, Depends(async_dep)]
) -> AsyncGenerator[str, None]:
    try:
        yield ASYNC_CM_DEPENDENCY_VALUE + "_" + async_dep
    finally:
        pass


@contextmanager
def sync_cm_dep(
    input: EmptyModel, ctx: Context, sync_dep: Annotated[str, Depends(sync_dep)]
) -> Generator[str, None, None]:
    try:
        yield SYNC_CM_DEPENDENCY_VALUE + "_" + sync_dep
    finally:
        pass


@contextmanager
def base_cm_dep(input: EmptyModel, ctx: Context) -> Generator[str, None, None]:
    try:
        yield CHAINED_CM_VALUE
    finally:
        pass


def chained_dep(
    input: EmptyModel, ctx: Context, base_cm: Annotated[str, Depends(base_cm_dep)]
) -> str:
    return "chained_" + base_cm


@asynccontextmanager
async def base_async_cm_dep(
    input: EmptyModel, ctx: Context
) -> AsyncGenerator[str, None]:
    try:
        yield CHAINED_ASYNC_CM_VALUE
    finally:
        pass


async def chained_async_dep(
    input: EmptyModel,
    ctx: Context,
    base_async_cm: Annotated[str, Depends(base_async_cm_dep)],
) -> str:
    return "chained_" + base_async_cm


# !!


class Output(BaseModel):
    sync_dep: str
    async_dep: str
    async_cm_dep: str
    sync_cm_dep: str
    chained_dep: str
    chained_async_dep: str


# > Inject dependencies
@hatchet.task()
async def async_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
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
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
    )


@hatchet.durable_task()
async def durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
    )


@hatchet.durable_task()
def durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
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
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
    )


@di_workflow.task()
def wf_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
    )


@di_workflow.durable_task()
async def wf_durable_async_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
    )


@di_workflow.durable_task()
def wf_durable_sync_task_with_dependencies(
    _i: EmptyModel,
    ctx: DurableContext,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
    async_cm_dep: Annotated[str, Depends(async_cm_dep)],
    sync_cm_dep: Annotated[str, Depends(sync_cm_dep)],
    chained_dep: Annotated[str, Depends(chained_dep)],
    chained_async_dep: Annotated[str, Depends(chained_async_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
        async_cm_dep=async_cm_dep,
        sync_cm_dep=sync_cm_dep,
        chained_dep=chained_dep,
        chained_async_dep=chained_async_dep,
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
            task_with_type_aliases,
        ],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
