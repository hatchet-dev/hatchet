from collections.abc import Callable
from typing import TYPE_CHECKING, Any, Generic, cast, get_type_hints

from hatchet_sdk.conditions import (
    Action,
    Condition,
    OrGroup,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
    flatten_conditions,
)
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.shared.condition_pb2 import TaskConditions
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    CreateTaskOpts,
    CreateTaskRateLimit,
    DesiredWorkerLabels,
)
from hatchet_sdk.runnables.types import (
    ConcurrencyExpression,
    R,
    StepType,
    TWorkflowInput,
    is_async_fn,
    is_durable_sync_fn,
    is_sync_fn,
)
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import (
    AwaitableLike,
    CoroutineLike,
    TaskIOValidator,
    is_basemodel_subclass,
)

if TYPE_CHECKING:
    from hatchet_sdk.runnables.workflow import Workflow


class Task(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        _fn: (
            Callable[[TWorkflowInput, Context], R | CoroutineLike[R]]
            | Callable[[TWorkflowInput, Context], AwaitableLike[R]]
            | (
                Callable[[TWorkflowInput, DurableContext], R | CoroutineLike[R]]
                | Callable[[TWorkflowInput, DurableContext], AwaitableLike[R]]
            )
        ),
        is_durable: bool,
        type: StepType,
        workflow: "Workflow[TWorkflowInput]",
        name: str,
        execution_timeout: Duration,
        schedule_timeout: Duration,
        parents: "list[Task[TWorkflowInput, Any]] | None",
        retries: int,
        rate_limits: list[CreateTaskRateLimit] | None,
        desired_worker_labels: dict[str, DesiredWorkerLabels] | None,
        backoff_factor: float | None,
        backoff_max_seconds: int | None,
        concurrency: list[ConcurrencyExpression] | None,
        wait_for: list[Condition | OrGroup] | None,
        skip_if: list[Condition | OrGroup] | None,
        cancel_if: list[Condition | OrGroup] | None,
    ) -> None:
        self.is_durable = is_durable

        self.fn = _fn
        self.is_async_function = is_async_fn(self.fn)  # type: ignore

        self.workflow = workflow

        self.type = type
        self.execution_timeout = execution_timeout
        self.schedule_timeout = schedule_timeout
        self.name = name
        self.parents = parents or []
        self.retries = retries
        self.rate_limits = rate_limits or []
        self.desired_worker_labels = desired_worker_labels or {}
        self.backoff_factor = backoff_factor
        self.backoff_max_seconds = backoff_max_seconds
        self.concurrency = concurrency or []

        self.wait_for = flatten_conditions(wait_for or [])
        self.skip_if = flatten_conditions(skip_if or [])
        self.cancel_if = flatten_conditions(cancel_if or [])

        return_type = get_type_hints(_fn).get("return")

        self.validators: TaskIOValidator = TaskIOValidator(
            workflow_input=workflow.config.input_validator,
            step_output=return_type if is_basemodel_subclass(return_type) else None,
        )

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
                return fn(workflow_input, ctx)

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

    def to_proto(self, service_name: str) -> CreateTaskOpts:
        return CreateTaskOpts(
            readable_id=self.name,
            action=service_name + ":" + self.name,
            timeout=timedelta_to_expr(self.execution_timeout),
            inputs="{}",
            parents=[p.name for p in self.parents],
            retries=self.retries,
            rate_limits=self.rate_limits,
            worker_labels=self.desired_worker_labels,
            backoff_factor=self.backoff_factor,
            backoff_max_seconds=self.backoff_max_seconds,
            concurrency=[t.to_proto() for t in self.concurrency],
            conditions=self._conditions_to_proto(),
            schedule_timeout=timedelta_to_expr(self.schedule_timeout),
        )

    def _assign_action(self, condition: Condition, action: Action) -> Condition:
        condition.base.action = action

        return condition

    def _conditions_to_proto(self) -> TaskConditions:
        wait_for_conditions = [
            self._assign_action(w, Action.QUEUE) for w in self.wait_for
        ]

        cancel_if_conditions = [
            self._assign_action(c, Action.CANCEL) for c in self.cancel_if
        ]
        skip_if_conditions = [self._assign_action(s, Action.SKIP) for s in self.skip_if]

        conditions = wait_for_conditions + cancel_if_conditions + skip_if_conditions

        if len({c.base.readable_data_key for c in conditions}) != len(
            [c.base.readable_data_key for c in conditions]
        ):
            raise ValueError("Conditions must have unique readable data keys.")

        user_events = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, UserEventCondition)
        ]
        parent_overrides = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, ParentCondition)
        ]
        sleep_conditions = [
            c.to_proto(self.workflow.client.config)
            for c in conditions
            if isinstance(c, SleepCondition)
        ]

        return TaskConditions(
            parent_override_conditions=parent_overrides,
            sleep_conditions=sleep_conditions,
            user_event_conditions=user_events,
        )
