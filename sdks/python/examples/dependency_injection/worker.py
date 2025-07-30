# > Simple

from typing import Annotated

from pydantic import BaseModel

from hatchet_sdk import Context, Depends, EmptyModel, Hatchet

hatchet = Hatchet(debug=True)


async def async_dep() -> str:
    return "async_dependency_value"


def sync_dep() -> str:
    return "sync_dependency_value"


class Output(BaseModel):
    sync_dep: str
    async_dep: str


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


def main() -> None:
    worker = hatchet.worker(
        "dependency-injection-worker",
        workflows=[async_task_with_dependencies, sync_task_with_dependencies],
    )
    worker.start()


# !!

if __name__ == "__main__":
    main()
