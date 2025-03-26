import asyncio
from datetime import timedelta
from enum import Enum
from typing import Any, Awaitable, Callable, ParamSpec, Type, TypeGuard, TypeVar, Union

from pydantic import BaseModel, ConfigDict, Field, StrictInt, model_validator

from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.utils.typing import JSONSerializableMapping

ValidTaskReturnType = Union[BaseModel, JSONSerializableMapping, None]

R = TypeVar("R", bound=Union[ValidTaskReturnType, Awaitable[ValidTaskReturnType]])
P = ParamSpec("P")


DEFAULT_EXECUTION_TIMEOUT = timedelta(seconds=60)
DEFAULT_SCHEDULE_TIMEOUT = timedelta(minutes=5)
DEFAULT_PRIORITY = 1


class EmptyModel(BaseModel):
    model_config = ConfigDict(extra="allow", frozen=True)


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


class TaskDefaults(BaseModel):
    schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT
    execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT
    priority: StrictInt = Field(gt=0, lt=4, default=DEFAULT_PRIORITY)


class WorkflowConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", arbitrary_types_allowed=True)

    name: str
    description: str | None = None
    version: str | None = None
    on_events: list[str] = Field(default_factory=list)
    on_crons: list[str] = Field(default_factory=list)
    sticky: StickyStrategy | None = None
    concurrency: ConcurrencyExpression | None = None
    input_validator: Type[BaseModel] = EmptyModel

    task_defaults: TaskDefaults = TaskDefaults()

    @model_validator(mode="after")
    def validate_concurrency_expression(self) -> "WorkflowConfig":
        if not self.concurrency:
            return self

        expr = self.concurrency.expression

        if not expr.startswith("input."):
            return self

        _, field = expr.split(".", maxsplit=2)

        if field not in self.input_validator.model_fields.keys():
            raise ValueError(
                f"The concurrency expression provided relies on the `{field}` field, which was not present in `{self.input_validator.__name__}`."
            )

        return self


class StepType(str, Enum):
    DEFAULT = "default"
    ON_FAILURE = "on_failure"
    ON_SUCCESS = "on_success"


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


DurableAsyncFunc = Callable[[TWorkflowInput, DurableContext], Awaitable[R]]
DurableSyncFunc = Callable[[TWorkflowInput, DurableContext], R]
DurableTaskFunc = Union[
    DurableAsyncFunc[TWorkflowInput, R], DurableSyncFunc[TWorkflowInput, R]
]


def is_durable_async_fn(
    fn: Callable[..., Any]
) -> TypeGuard[DurableAsyncFunc[TWorkflowInput, R]]:
    return asyncio.iscoroutinefunction(fn)


def is_durable_sync_fn(
    fn: DurableTaskFunc[TWorkflowInput, R]
) -> TypeGuard[DurableSyncFunc[TWorkflowInput, R]]:
    return not asyncio.iscoroutinefunction(fn)
