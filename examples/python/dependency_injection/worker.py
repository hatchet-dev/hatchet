# > Simple

from typing import Annotated

from pydantic import BaseModel

from hatchet_sdk import Context, Depends, DurableContext, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


# > Declare dependencies
async def async_dep() -> str:
    return "async_dependency_value"


def sync_dep() -> str:
    return "sync_dependency_value"




class Output(BaseModel):
    sync_dep: str
    async_dep: str


# > Inject dependencies
@hatchet.task()
async def async_task_with_dependencies(
    input: EmptyModel,
    ctx: Context,
    async_dep: Annotated[str, Depends(async_dep)],
    sync_dep: Annotated[str, Depends(sync_dep)],
) -> Output:
    return Output(
        sync_dep=sync_dep,
        async_dep=async_dep,
    )




@hatchet.task()
def sync_task_with_dependencies(
    input: EmptyModel,
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
    input: EmptyModel,
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
    input: EmptyModel,
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
    input: EmptyModel,
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
    input: EmptyModel,
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
    input: EmptyModel,
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
    input: EmptyModel,
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
            durable_async_task_with_dependencies,
            di_workflow,
        ],
    )
    worker.start()



if __name__ == "__main__":
    main()
