from datetime import timedelta
from typing import TYPE_CHECKING, Any, Awaitable, Callable, Generic

from hatchet_sdk.context.context import Context
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    CreateTaskRateLimit,
    DesiredWorkerLabels,
)
from hatchet_sdk.runnables.types import (
    ConcurrencyLimitStrategy,
    R,
    StepType,
    TWorkflowInput,
    is_async_fn,
    is_sync_fn,
)
from hatchet_sdk.waits import Condition, OrGroup

if TYPE_CHECKING:
    from hatchet_sdk.runnables.workflow import Workflow


class Task(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        fn: (
            Callable[[TWorkflowInput, Context], R]
            | Callable[[TWorkflowInput, Context], Awaitable[R]]
        ),
        type: StepType,
        workflow: "Workflow[TWorkflowInput]",
        name: str,
        timeout: timedelta | str = timedelta(minutes=60),
        parents: "list[Task[TWorkflowInput, Any]]" = [],
        retries: int = 0,
        rate_limits: list[CreateTaskRateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabels] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency__slots: int | None = None,
        concurrency__limit_strategy: ConcurrencyLimitStrategy | None = None,
        wait_for: list[Condition | OrGroup] = [],
        skip_if: list[Condition | OrGroup] = [],
        cancel_if: list[Condition | OrGroup] = [],
    ) -> None:
        self.fn = fn
        self.is_async_function = is_async_fn(fn)
        self.workflow = workflow

        self.type = type
        self.timeout = timeout
        self.name = name
        self.parents = parents
        self.retries = retries
        self.rate_limits = rate_limits
        self.desired_worker_labels = desired_worker_labels
        self.backoff_factor = backoff_factor
        self.backoff_max_seconds = backoff_max_seconds
        self.concurrency__slots = concurrency__slots
        self.concurrency__limit_strategy = concurrency__limit_strategy

        self.wait_for = self._flatten_conditions(wait_for)
        self.skip_if = self._flatten_conditions(skip_if)
        self.cancel_if = self._flatten_conditions(cancel_if)

    def _flatten_conditions(
        self, conditions: list[Condition | OrGroup]
    ) -> list[Condition]:
        flattened: list[Condition] = []

        for condition in conditions:
            if isinstance(condition, OrGroup):
                flattened.extend(
                    [
                        c.model_copy(update={"or_group_id": condition.or_group_id})
                        for c in condition.conditions
                    ]
                )
            else:
                flattened.append(condition)

        return flattened

    def call(self, ctx: Context) -> R:
        if self.is_async_function:
            raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

        sync_fn = self.fn
        workflow_input = self.workflow._get_workflow_input(ctx)

        if is_sync_fn(sync_fn):
            return sync_fn(workflow_input, ctx)

        raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

    async def aio_call(self, ctx: Context) -> R:
        if not self.is_async_function:
            raise TypeError(
                f"{self.name} is not an async function. Use `call` instead."
            )

        async_fn = self.fn
        workflow_input = self.workflow._get_workflow_input(ctx)

        if is_async_fn(async_fn):
            return await async_fn(workflow_input, ctx)

        raise TypeError(f"{self.name} is not an async function. Use `call` instead.")
