from typing import TYPE_CHECKING, Any, Awaitable, Callable, Generic

from hatchet_sdk.context.context import Context
from hatchet_sdk.contracts.workflows_pb2 import CreateStepRateLimit, DesiredWorkerLabels
from hatchet_sdk.runnables.types import (
    ConcurrencyLimitStrategy,
    R,
    StepType,
    TWorkflowInput,
    is_async_fn,
    is_sync_fn,
)

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
        name: str = "",
        timeout: str = "60m",
        parents: "list[Task[TWorkflowInput, Any]]" = [],
        retries: int = 0,
        rate_limits: list[CreateStepRateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabels] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency__slots: int | None = None,
        concurrency__limit_strategy: ConcurrencyLimitStrategy | None = None,
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

    def call(self, ctx: Context) -> R:
        if self.is_async_function:
            raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

        sync_fn = self.fn
        workflow_input = self.workflow.get_workflow_input(ctx)

        if is_sync_fn(sync_fn):
            return sync_fn(workflow_input, ctx)

        raise TypeError(f"{self.name} is not a sync function. Use `acall` instead.")

    async def aio_call(self, ctx: Context) -> R:
        if not self.is_async_function:
            raise TypeError(
                f"{self.name} is not an async function. Use `call` instead."
            )

        async_fn = self.fn
        workflow_input = self.workflow.get_workflow_input(ctx)

        if is_async_fn(async_fn):
            return await async_fn(workflow_input, ctx)

        raise TypeError(f"{self.name} is not an async function. Use `call` instead.")
