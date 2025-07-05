import functools
from typing import (
    Any,
    Callable,
    Protocol,
    Type,
    TypeVar,
    Union,
    cast,
    get_type_hints,
    runtime_checkable,
)

from pydantic import BaseModel

from hatchet_sdk.contracts.workflows_pb2 import (
    CreateWorkflowJobOpts,
    CreateWorkflowStepOpts,
    CreateWorkflowVersionOpts,
    StickyStrategy,
    WorkflowConcurrencyOpts,
    WorkflowKind,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.v0 import ConcurrencyLimitStrategy
from hatchet_sdk.v0.utils.typing import is_basemodel_subclass


class WorkflowStepProtocol(Protocol):
    def __call__(self, *args: Any, **kwargs: Any) -> Any: ...

    __name__: str

    _step_name: str
    _step_timeout: str | None
    _step_parents: list[str]
    _step_retries: int | None
    _step_rate_limits: list[str] | None
    _step_desired_worker_labels: dict[str, str]
    _step_backoff_factor: float | None
    _step_backoff_max_seconds: int | None

    _concurrency_fn_name: str
    _concurrency_max_runs: int | None
    _concurrency_limit_strategy: str | None

    _on_failure_step_name: str
    _on_failure_step_timeout: str | None
    _on_failure_step_retries: int
    _on_failure_step_rate_limits: list[str] | None
    _on_failure_step_backoff_factor: float | None
    _on_failure_step_backoff_max_seconds: int | None


StepsType = list[tuple[str, WorkflowStepProtocol]]

T = TypeVar("T")
TW = TypeVar("TW", bound="WorkflowInterface")


class ConcurrencyExpression:
    """
    Defines concurrency limits for a workflow using a CEL expression.

    Args:
        expression (str): CEL expression to determine concurrency grouping. (i.e. "input.user_id")
        max_runs (int): Maximum number of concurrent workflow runs.
        limit_strategy (ConcurrencyLimitStrategy): Strategy for handling limit violations.

    Example:
        ConcurrencyExpression("input.user_id", 5, ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS)
    """

    def __init__(
        self, expression: str, max_runs: int, limit_strategy: ConcurrencyLimitStrategy
    ):
        self.expression = expression
        self.max_runs = max_runs
        self.limit_strategy = limit_strategy


@runtime_checkable
class WorkflowInterface(Protocol):
    def get_name(self, namespace: str) -> str: ...

    def get_actions(self, namespace: str) -> list[tuple[str, Callable[..., Any]]]: ...

    def get_create_opts(self, namespace: str) -> Any: ...

    on_events: list[str] | None
    on_crons: list[str] | None
    name: str
    version: str
    timeout: str
    schedule_timeout: str
    sticky: Union[StickyStrategy.Value, None]  # type: ignore[name-defined]
    default_priority: int | None
    concurrency_expression: ConcurrencyExpression | None
    input_validator: Type[BaseModel] | None


class WorkflowMeta(type):
    def __new__(
        cls: Type["WorkflowMeta"],
        name: str,
        bases: tuple[type, ...],
        attrs: dict[str, Any],
    ) -> "WorkflowMeta":
        def _create_steps_actions_list(name: str) -> StepsType:
            return [
                (getattr(func, name), attrs.pop(func_name))
                for func_name, func in list(attrs.items())
                if hasattr(func, name)
            ]

        concurrencyActions = _create_steps_actions_list("_concurrency_fn_name")
        steps = _create_steps_actions_list("_step_name")

        onFailureSteps = _create_steps_actions_list("_on_failure_step_name")

        # Define __init__ and get_step_order methods
        original_init = attrs.get("__init__")  # Get the original __init__ if it exists

        def __init__(self: TW, *args: Any, **kwargs: Any) -> None:
            if original_init:
                original_init(self, *args, **kwargs)  # Call original __init__

        def get_service_name(namespace: str) -> str:
            return f"{namespace}{name.lower()}"

        @functools.cache
        def get_actions(self: TW, namespace: str) -> StepsType:
            serviceName = get_service_name(namespace)

            func_actions = [
                (serviceName + ":" + func_name, func) for func_name, func in steps
            ]
            concurrency_actions = [
                (serviceName + ":" + func_name, func)
                for func_name, func in concurrencyActions
            ]
            onFailure_actions = [
                (serviceName + ":" + func_name, func)
                for func_name, func in onFailureSteps
            ]

            return func_actions + concurrency_actions + onFailure_actions

        # Add these methods and steps to class attributes
        attrs["__init__"] = __init__
        attrs["get_actions"] = get_actions

        for step_name, step_func in steps:
            attrs[step_name] = step_func

        def get_name(self: TW, namespace: str) -> str:
            return namespace + cast(str, attrs["name"])

        attrs["get_name"] = get_name

        cron_triggers = attrs["on_crons"]
        version = attrs["version"]
        schedule_timeout = attrs["schedule_timeout"]
        sticky = attrs["sticky"]
        default_priority = attrs["default_priority"]

        @functools.cache
        def get_create_opts(self: TW, namespace: str) -> CreateWorkflowVersionOpts:
            serviceName = get_service_name(namespace)
            name = self.get_name(namespace)
            event_triggers = [namespace + event for event in attrs["on_events"]]
            createStepOpts: list[CreateWorkflowStepOpts] = [
                CreateWorkflowStepOpts(
                    readable_id=step_name,
                    action=serviceName + ":" + step_name,
                    timeout=func._step_timeout or "60s",
                    inputs="{}",
                    parents=[x for x in func._step_parents],
                    retries=func._step_retries,
                    rate_limits=func._step_rate_limits,  # type: ignore[arg-type]
                    worker_labels=func._step_desired_worker_labels,  # type: ignore[arg-type]
                    backoff_factor=func._step_backoff_factor,
                    backoff_max_seconds=func._step_backoff_max_seconds,
                )
                for step_name, func in steps
            ]

            concurrency: WorkflowConcurrencyOpts | None = None

            if len(concurrencyActions) > 0:
                action = concurrencyActions[0]

                concurrency = WorkflowConcurrencyOpts(
                    action=serviceName + ":" + action[0],
                    max_runs=action[1]._concurrency_max_runs,
                    limit_strategy=action[1]._concurrency_limit_strategy,
                )

            if self.concurrency_expression:
                concurrency = WorkflowConcurrencyOpts(
                    expression=self.concurrency_expression.expression,
                    max_runs=self.concurrency_expression.max_runs,
                    limit_strategy=self.concurrency_expression.limit_strategy,
                )

            if len(concurrencyActions) > 0 and self.concurrency_expression:
                raise ValueError(
                    "Error: Both concurrencyActions and concurrency_expression are defined. Please use only one concurrency configuration method."
                )

            on_failure_job: CreateWorkflowJobOpts | None = None

            if len(onFailureSteps) > 0:
                func_name, func = onFailureSteps[0]
                on_failure_job = CreateWorkflowJobOpts(
                    name=name + "-on-failure",
                    steps=[
                        CreateWorkflowStepOpts(
                            readable_id=func_name,
                            action=serviceName + ":" + func_name,
                            timeout=func._on_failure_step_timeout or "60s",
                            inputs="{}",
                            parents=[],
                            retries=func._on_failure_step_retries,
                            rate_limits=func._on_failure_step_rate_limits,  # type: ignore[arg-type]
                            backoff_factor=func._on_failure_step_backoff_factor,
                            backoff_max_seconds=func._on_failure_step_backoff_max_seconds,
                        )
                    ],
                )

            validated_priority = (
                max(1, min(3, default_priority)) if default_priority else None
            )
            if validated_priority != default_priority:
                logger.warning(
                    "Warning: Default Priority Must be between 1 and 3 -- inclusively. Adjusted to be within the range."
                )

            return CreateWorkflowVersionOpts(
                name=name,
                kind=WorkflowKind.DAG,
                version=version,
                event_triggers=event_triggers,
                cron_triggers=cron_triggers,
                schedule_timeout=schedule_timeout,
                sticky=sticky,
                jobs=[
                    CreateWorkflowJobOpts(
                        name=name,
                        steps=createStepOpts,
                    )
                ],
                on_failure_job=on_failure_job,
                concurrency=concurrency,
                default_priority=validated_priority,
            )

        attrs["get_create_opts"] = get_create_opts

        return super(WorkflowMeta, cls).__new__(cls, name, bases, attrs)
