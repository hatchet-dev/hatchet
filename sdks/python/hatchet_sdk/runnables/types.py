import asyncio
from datetime import timedelta
from enum import Enum
from typing import Awaitable, Callable, ParamSpec, Type, TypeGuard, TypeVar, Union

from pydantic import BaseModel, ConfigDict, Field, model_validator

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
    on_events: list[str] = Field(default_factory=list)
    on_crons: list[str] = Field(default_factory=list)
    version: str | None = None
    schedule_timeout: timedelta = timedelta(minutes=5)
    sticky: StickyStrategy | None = None
    default_priority: int = 1
    concurrency: ConcurrencyExpression | None = None
    input_validator: Type[BaseModel] = EmptyModel

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
                f"The concurrency expression provided relies on the `{field}` field, which was not present in the `input_validator`."
            )

        return self


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
