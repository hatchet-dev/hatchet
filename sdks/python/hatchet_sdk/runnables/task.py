from typing import (
    TYPE_CHECKING,
    Any,
    Awaitable,
    Callable,
    Generic,
    TypeVar,
    Union,
    cast,
)

from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    CreateTaskRateLimit,
    DesiredWorkerLabels,
)
from hatchet_sdk.runnables.types import (
    DEFAULT_EXECUTION_TIMEOUT,
    DEFAULT_SCHEDULE_TIMEOUT,
    ConcurrencyExpression,
    R,
    StepType,
    TWorkflowInput,
    is_async_fn,
    is_durable_sync_fn,
    is_sync_fn,
)
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.waits import Condition, OrGroup

if TYPE_CHECKING:
    from hatchet_sdk.runnables.workflow import Workflow


T = TypeVar("T")


def fall_back_to_default(value: T, default: T, fallback_value: T) -> T:
    ## If the value is not the default, it's set
    if value != default:
        return value

    ## Otherwise, it's unset, so return the fallback value
    return fallback_value


class Task(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        _fn: Union[
            Callable[[TWorkflowInput, Context], R]
            | Callable[[TWorkflowInput, Context], Awaitable[R]],
            Callable[[TWorkflowInput, DurableContext], R]
            | Callable[[TWorkflowInput, DurableContext], Awaitable[R]],
        ],
        is_durable: bool,
        type: StepType,
        workflow: "Workflow[TWorkflowInput]",
        name: str,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        parents: "list[Task[TWorkflowInput, Any]]" = [],
        retries: int = 0,
        rate_limits: list[CreateTaskRateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabels] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: list[ConcurrencyExpression] = [],
        wait_for: list[Condition | OrGroup] = [],
        skip_if: list[Condition | OrGroup] = [],
        cancel_if: list[Condition | OrGroup] = [],
    ) -> None:
        self.is_durable = is_durable

        self.fn = _fn
        self.is_async_function = is_async_fn(self.fn)  # type: ignore

        self.workflow = workflow

        self.type = type
        self.execution_timeout = fall_back_to_default(
            execution_timeout, DEFAULT_EXECUTION_TIMEOUT, DEFAULT_EXECUTION_TIMEOUT
        )
        self.schedule_timeout = fall_back_to_default(
            schedule_timeout, DEFAULT_SCHEDULE_TIMEOUT, DEFAULT_SCHEDULE_TIMEOUT
        )
        self.name = name
        self.parents = parents
        self.retries = retries
        self.rate_limits = rate_limits
        self.desired_worker_labels = desired_worker_labels
        self.backoff_factor = backoff_factor
        self.backoff_max_seconds = backoff_max_seconds
        self.concurrency = concurrency

        self.wait_for = self._flatten_conditions(wait_for)
        self.skip_if = self._flatten_conditions(skip_if)
        self.cancel_if = self._flatten_conditions(cancel_if)

    def _flatten_conditions(
        self, conditions: list[Condition | OrGroup]
    ) -> list[Condition]:
        flattened: list[Condition] = []

        for condition in conditions:
            if isinstance(condition, OrGroup):
                for or_condition in condition.conditions:
                    or_condition.base.or_group_id = condition.or_group_id

                flattened.extend(condition.conditions)
            else:
                flattened.append(condition)

        return flattened

    def call(self, ctx: Context | DurableContext) -> R:
        if self.is_async_function:
            raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

        workflow_input = self.workflow._get_workflow_input(ctx)

        if self.is_durable:
            fn = cast(Callable[[TWorkflowInput, DurableContext], R], self.fn)
            if is_durable_sync_fn(fn):
                return fn(workflow_input, cast(DurableContext, ctx))
        else:
            fn = cast(Callable[[TWorkflowInput, Context], R], self.fn)
            if is_sync_fn(fn):
                return fn(workflow_input, cast(Context, ctx))

        raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

    async def aio_call(self, ctx: Context | DurableContext) -> R:
        if not self.is_async_function:
            raise TypeError(
                f"{self.name} is not an async function. Use `call` instead."
            )

        workflow_input = self.workflow._get_workflow_input(ctx)

        if is_async_fn(self.fn):  # type: ignore
            return await self.fn(workflow_input, cast(Context, ctx))  # type: ignore

        raise TypeError(f"{self.name} is not an async function. Use `call` instead.")
