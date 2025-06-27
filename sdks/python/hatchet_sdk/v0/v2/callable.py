import asyncio
from typing import (
    Any,
    Callable,
    Dict,
    Generic,
    List,
    Optional,
    TypedDict,
    TypeVar,
    Union,
)

from hatchet_sdk.contracts.workflows_pb2 import (  # type: ignore[attr-defined]
    CreateStepRateLimit,
    CreateWorkflowJobOpts,
    CreateWorkflowStepOpts,
    CreateWorkflowVersionOpts,
    DesiredWorkerLabels,
    StickyStrategy,
    WorkflowConcurrencyOpts,
    WorkflowKind,
)
from hatchet_sdk.logger import logger
from hatchet_sdk.v0.clients.admin import ChildTriggerWorkflowOptions
from hatchet_sdk.v0.context.context import Context
from hatchet_sdk.v0.labels import DesiredWorkerLabel
from hatchet_sdk.v0.rate_limit import RateLimit
from hatchet_sdk.v0.v2.concurrency import ConcurrencyFunction
from hatchet_sdk.v0.workflow_run import RunRef

T = TypeVar("T")


class HatchetCallable(Generic[T]):
    def __init__(
        self,
        func: Callable[[Context], T],
        durable: bool = False,
        name: str = "",
        auto_register: bool = True,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str = "",
        timeout: str = "60m",
        schedule_timeout: str = "5m",
        sticky: StickyStrategy = None,
        retries: int = 0,
        rate_limits: List[RateLimit] | None = None,
        concurrency: ConcurrencyFunction | None = None,
        on_failure: Union["HatchetCallable[T]", None] = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        default_priority: int | None = None,
    ):
        self.func = func

        on_events = on_events or []
        on_crons = on_crons or []

        limits = None
        if rate_limits:
            limits = [rate_limit._req for rate_limit in rate_limits or []]

        self.function_desired_worker_labels = {}

        for key, d in desired_worker_labels.items():
            value = d["value"] if "value" in d else None
            self.function_desired_worker_labels[key] = DesiredWorkerLabels(
                strValue=str(value) if not isinstance(value, int) else None,
                intValue=value if isinstance(value, int) else None,
                required=d["required"] if "required" in d else None,
                weight=d["weight"] if "weight" in d else None,
                comparator=d["comparator"] if "comparator" in d else None,
            )
        self.sticky = sticky
        self.default_priority = default_priority
        self.durable = durable
        self.function_name = name.lower() or str(func.__name__).lower()
        self.function_version = version
        self.function_on_events = on_events
        self.function_on_crons = on_crons
        self.function_timeout = timeout
        self.function_schedule_timeout = schedule_timeout
        self.function_retries = retries
        self.function_rate_limits = limits
        self.function_concurrency = concurrency
        self.function_on_failure = on_failure
        self.function_namespace = "default"
        self.function_auto_register = auto_register

        self.is_coroutine = False

        if asyncio.iscoroutinefunction(func):
            self.is_coroutine = True

    def __call__(self, context: Context) -> T:
        return self.func(context)

    def with_namespace(self, namespace: str) -> None:
        if namespace is not None and namespace != "":
            self.function_namespace = namespace
            self.function_name = namespace + self.function_name

    def to_workflow_opts(self) -> CreateWorkflowVersionOpts:
        kind: WorkflowKind = WorkflowKind.FUNCTION

        if self.durable:
            kind = WorkflowKind.DURABLE

        on_failure_job: CreateWorkflowJobOpts | None = None

        if self.function_on_failure is not None:
            on_failure_job = CreateWorkflowJobOpts(
                name=self.function_name + "-on-failure",
                steps=[
                    self.function_on_failure.to_step(),
                ],
            )

        concurrency: WorkflowConcurrencyOpts | None = None

        if self.function_concurrency is not None:
            self.function_concurrency.set_namespace(self.function_namespace)
            concurrency = WorkflowConcurrencyOpts(
                action=self.function_concurrency.get_action_name(),
                max_runs=self.function_concurrency.max_runs,
                limit_strategy=self.function_concurrency.limit_strategy,
            )

        validated_priority = (
            max(1, min(3, self.default_priority)) if self.default_priority else None
        )
        if validated_priority != self.default_priority:
            logger.warning(
                "Warning: Default Priority Must be between 1 and 3 -- inclusively. Adjusted to be within the range."
            )

        return CreateWorkflowVersionOpts(
            name=self.function_name,
            kind=kind,
            version=self.function_version,
            event_triggers=self.function_on_events,
            cron_triggers=self.function_on_crons,
            schedule_timeout=self.function_schedule_timeout,
            sticky=self.sticky,
            on_failure_job=on_failure_job,
            concurrency=concurrency,
            jobs=[
                CreateWorkflowJobOpts(
                    name=self.function_name,
                    steps=[
                        self.to_step(),
                    ],
                )
            ],
            default_priority=validated_priority,
        )

    def to_step(self) -> CreateWorkflowStepOpts:
        return CreateWorkflowStepOpts(
            readable_id=self.function_name,
            action=self.get_action_name(),
            timeout=self.function_timeout,
            inputs="{}",
            parents=[],
            retries=self.function_retries,
            rate_limits=self.function_rate_limits,
            worker_labels=self.function_desired_worker_labels,
        )

    def get_action_name(self) -> str:
        return self.function_namespace + ":" + self.function_name


class DurableContext(Context):
    def run(
        self,
        function: str | Callable[[Context], Any],
        input: dict[Any, Any] = {},
        key: str | None = None,
        options: ChildTriggerWorkflowOptions | None = None,
    ) -> "RunRef[T]":
        worker_id = self.worker.id()

        workflow_name = function

        if not isinstance(function, str):
            workflow_name = function.function_name

        # if (
        #     options is not None
        #     and "sticky" in options
        #     and options["sticky"] == True
        #     and not self.worker.has_workflow(workflow_name)
        # ):
        #     raise Exception(
        #         f"cannot run with sticky: workflow {workflow_name} is not registered on the worker"
        #     )

        trigger_options = self._prepare_workflow_options(key, options, worker_id)

        return self.admin_client.run(function, input, trigger_options)
