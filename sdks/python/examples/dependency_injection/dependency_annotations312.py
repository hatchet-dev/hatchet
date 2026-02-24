from hatchet_sdk.runnables.task import Depends
from hatchet_sdk import Context
from hatchet_sdk.runnables.types import EmptyModel
from typing import Annotated, TypeAlias


async def async_dep(input: EmptyModel, ctx: Context) -> bool:
    return True


def sync_dep(input: EmptyModel, ctx: Context) -> bool:
    return True


AsyncDepNoTypeAlias = Annotated[bool, Depends(async_dep)]
AsyncDepTypeAlias: TypeAlias = Annotated[bool, Depends(async_dep)]
type AsyncDepTypeSyntax = Annotated[bool, Depends(async_dep)]

SyncDepNoTypeAlias = Annotated[bool, Depends(sync_dep)]
SyncDepTypeAlias: TypeAlias = Annotated[bool, Depends(sync_dep)]
type SyncDepTypeSyntax = Annotated[bool, Depends(sync_dep)]
