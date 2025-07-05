import asyncio
import json
from collections.abc import Callable
from enum import Enum
from typing import Any, ParamSpec, TypeGuard, TypeVar

from pydantic import BaseModel, ConfigDict, Field

from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.workflows_pb2 import Concurrency
from hatchet_sdk.contracts.v1.workflows_pb2 import DefaultFilter as DefaultFilterProto
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.utils.typing import AwaitableLike, JSONSerializableMapping

ValidTaskReturnType = BaseModel | JSONSerializableMapping | None

R = TypeVar("R", bound=ValidTaskReturnType)
P = ParamSpec("P")


class EmptyModel(BaseModel):
    model_config = ConfigDict(extra="allow", frozen=True)


class StickyStrategy(str, Enum):
    SOFT = "SOFT"
    HARD = "HARD"


class ConcurrencyLimitStrategy(str, Enum):
    CANCEL_IN_PROGRESS = "CANCEL_IN_PROGRESS"
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

    def to_proto(self) -> Concurrency:
        return Concurrency(
            expression=self.expression,
            max_runs=self.max_runs,
            limit_strategy=self.limit_strategy,
        )


TWorkflowInput = TypeVar("TWorkflowInput", bound=BaseModel)


class TaskDefaults(BaseModel):
    schedule_timeout: Duration | None = None
    execution_timeout: Duration | None = None
    priority: int | None = Field(gt=0, lt=4, default=None)
    retries: int | None = None
    backoff_factor: float | None = None
    backoff_max_seconds: int | None = None


class DefaultFilter(BaseModel):
    expression: str
    scope: str
    payload: JSONSerializableMapping = Field(default_factory=dict)

    def to_proto(self) -> DefaultFilterProto:
        payload_json = json.dumps(self.payload, default=str)

        return DefaultFilterProto(
            expression=self.expression,
            scope=self.scope,
            payload=payload_json.encode("utf-8"),
        )


class WorkflowConfig(BaseModel):
    model_config = ConfigDict(extra="forbid", arbitrary_types_allowed=True)

    name: str
    description: str | None = None
    version: str | None = None
    on_events: list[str] = Field(default_factory=list)
    on_crons: list[str] = Field(default_factory=list)
    sticky: StickyStrategy | None = None
    concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None
    input_validator: type[BaseModel] = EmptyModel
    default_priority: int | None = None

    task_defaults: TaskDefaults = TaskDefaults()
    default_filters: list[DefaultFilter] = Field(default_factory=list)


class StepType(str, Enum):
    DEFAULT = "default"
    ON_FAILURE = "on_failure"
    ON_SUCCESS = "on_success"


AsyncFunc = Callable[[TWorkflowInput, Context], AwaitableLike[R]]
SyncFunc = Callable[[TWorkflowInput, Context], R]
TaskFunc = AsyncFunc[TWorkflowInput, R] | SyncFunc[TWorkflowInput, R]


def is_async_fn(
    fn: TaskFunc[TWorkflowInput, R],
) -> TypeGuard[AsyncFunc[TWorkflowInput, R]]:
    return asyncio.iscoroutinefunction(fn)


def is_sync_fn(
    fn: TaskFunc[TWorkflowInput, R],
) -> TypeGuard[SyncFunc[TWorkflowInput, R]]:
    return not asyncio.iscoroutinefunction(fn)


DurableAsyncFunc = Callable[[TWorkflowInput, DurableContext], AwaitableLike[R]]
DurableSyncFunc = Callable[[TWorkflowInput, DurableContext], R]
DurableTaskFunc = (
    DurableAsyncFunc[TWorkflowInput, R] | DurableSyncFunc[TWorkflowInput, R]
)


def is_durable_async_fn(
    fn: Callable[..., Any],
) -> TypeGuard[DurableAsyncFunc[TWorkflowInput, R]]:
    return asyncio.iscoroutinefunction(fn)


def is_durable_sync_fn(
    fn: DurableTaskFunc[TWorkflowInput, R],
) -> TypeGuard[DurableSyncFunc[TWorkflowInput, R]]:
    return not asyncio.iscoroutinefunction(fn)
