import asyncio
from enum import Enum
from typing import Awaitable, Callable, ParamSpec, Type, TypeGuard, TypeVar, Union

from pydantic import BaseModel, ConfigDict

from hatchet_sdk.context.context import Context
from hatchet_sdk.utils.typing import JSONSerializableMapping

ValidTaskReturnType = Union[BaseModel, JSONSerializableMapping, None]

R = TypeVar("R", bound=Union[ValidTaskReturnType, Awaitable[ValidTaskReturnType]])
P = ParamSpec("P")


class EmptyModel(BaseModel):
    model_config = ConfigDict(extra="allow")


class StickyStrategy(str, Enum):
    SOFT = "SOFT"
    HARD = "HARD"


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
    DROP_NEWEST = "DROP_NEWEST"
    QUEUE_NEWEST = "QUEUE_NEWEST"
    GROUP_ROUND_ROBIN = "GROUP_ROUND_ROBIN"
    CANCEL_NEWEST = "CANCEL_NEWEST"


class ConcurrencyExpression(BaseModel):
    """
    Defines concurrency limits for a workflow using a CEL expression.
    Args:
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int): Maximum number of concurrent workflow runs.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.
    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)
    """

    expression: str
    max_runs: int
    limit_strategy: ConcurrencyLimitStrategy


TWorkflowInput = TypeVar("TWorkflowInput", bound=BaseModel)


class WorkflowConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", arbitrary_types_allowed=True)

    name: str = ""
    on_events: list[str] = []
    on_crons: list[str] = []
    version: str = ""
    timeout: str = "60m"
    schedule_timeout: str = "5m"
    sticky: StickyStrategy | None = None
    default_priority: int = 1
    concurrency: ConcurrencyExpression | None = None
    input_validator: Type[BaseModel] = EmptyModel


class StepType(str, Enum):
    DEFAULT = "default"
    CONCURRENCY = "concurrency"
    ON_FAILURE = "on_failure"


AsyncFunc = Callable[[TWorkflowInput, Context], Awaitable[R]]
SyncFunc = Callable[[TWorkflowInput, Context], R]
TaskFunc = Union[AsyncFunc[TWorkflowInput, R], SyncFunc[TWorkflowInput, R]]


def is_async_fn(
    fn: TaskFunc[TWorkflowInput, R]
) -> TypeGuard[AsyncFunc[TWorkflowInput, R]]:
    return asyncio.iscoroutinefunction(fn)


def is_sync_fn(
    fn: TaskFunc[TWorkflowInput, R]
) -> TypeGuard[SyncFunc[TWorkflowInput, R]]:
    return not asyncio.iscoroutinefunction(fn)
