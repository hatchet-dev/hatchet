from datetime import datetime
from typing import TYPE_CHECKING, Any, Callable, Generic, cast

from google.protobuf import timestamp_pb2
from pydantic import BaseModel

from hatchet_sdk.clients.admin import (
    ScheduleTriggerWorkflowOptions,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.context.context import Context
from hatchet_sdk.contracts.workflows_pb2 import (
    ConcurrencyLimitStrategy as ConcurrencyLimitStrategyProto,
)
from hatchet_sdk.contracts.workflows_pb2 import (
    CreateWorkflowJobOpts,
    CreateWorkflowStepOpts,
    CreateWorkflowVersionOpts,
    DesiredWorkerLabels,
)
from hatchet_sdk.contracts.workflows_pb2 import StickyStrategy as StickyStrategyProto
from hatchet_sdk.contracts.workflows_pb2 import (
    WorkflowConcurrencyOpts,
    WorkflowKind,
    WorkflowVersion,
)
from hatchet_sdk.labels import DesiredWorkerLabel
from hatchet_sdk.logger import logger
from hatchet_sdk.rate_limit import RateLimit
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import R, StepType, TWorkflowInput, WorkflowConfig
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto, maybe_int_to_str
from hatchet_sdk.workflow_run import WorkflowRunRef

if TYPE_CHECKING:
    from hatchet_sdk import Hatchet


def transform_desired_worker_label(d: DesiredWorkerLabel) -> DesiredWorkerLabels:
    value = d.value
    return DesiredWorkerLabels(
        strValue=value if not isinstance(value, int) else None,
        intValue=value if isinstance(value, int) else None,
        required=d.required,
        weight=d.weight,
        comparator=d.comparator,  # type: ignore[arg-type]
    )


class TypedTriggerWorkflowRunConfig(BaseModel, Generic[TWorkflowInput]):
    input: TWorkflowInput
    options: TriggerWorkflowOptions


class Workflow(Generic[TWorkflowInput]):
    """
    A Hatchet workflow implementation base. This class should be inherited by all workflow implementations.

    A declaration is passed to the workflow using the `declaration` parameter. This declaration is used to
    define the workflow's configuration.
    """

    def __init__(self, config: WorkflowConfig, client: "Hatchet") -> None:
        self.config = config
        self.__default_tasks: list[Task[TWorkflowInput, Any]] = []
        self.__concurrency_actions: list[Task[TWorkflowInput, Any]] = []
        self.__on_failure_task: Task[TWorkflowInput, Any] | None = None
        self.client = client

    def get_service_name(self, namespace: str) -> str:
        return f"{namespace}{self.config.name.lower()}"

    @property
    def tasks(self) -> list[Task[TWorkflowInput, Any]]:
        tasks = self.__default_tasks + self.__concurrency_actions

        if self.__on_failure_task:
            tasks += [self.__on_failure_task]

        return tasks

    def create_action_name(
        self, namespace: str, step: Task[TWorkflowInput, Any]
    ) -> str:
        return self.get_service_name(namespace) + ":" + step.name

    def get_name(self, namespace: str) -> str:
        return namespace + self.config.name

    def _validate_concurrency_actions(
        self, service_name: str
    ) -> WorkflowConcurrencyOpts | None:
        if len(self.__concurrency_actions) > 0 and self.config.concurrency:
            raise ValueError(
                "Error: Both concurrencyActions and concurrency_expression are defined. Please use only one concurrency configuration method."
            )

        if len(self.__concurrency_actions) > 0:
            action = self.__concurrency_actions[0]

            return WorkflowConcurrencyOpts(
                action=service_name + ":" + action.name,
                max_runs=action.concurrency__slots,
                limit_strategy=maybe_int_to_str(
                    convert_python_enum_to_proto(
                        action.concurrency__limit_strategy,
                        ConcurrencyLimitStrategyProto,
                    )
                ),
            )

        if self.config.concurrency:
            return WorkflowConcurrencyOpts(
                expression=self.config.concurrency.expression,
                max_runs=self.config.concurrency.max_runs,
                limit_strategy=self.config.concurrency.limit_strategy,
            )

        return None

    def _validate_on_failure_task(
        self, name: str, service_name: str
    ) -> CreateWorkflowJobOpts | None:
        if not self.__on_failure_task:
            return None

        return CreateWorkflowJobOpts(
            name=name + "-on-failure",
            steps=[
                CreateWorkflowStepOpts(
                    readable_id=self.__on_failure_task.name,
                    action=service_name + ":" + self.__on_failure_task.name,
                    timeout=self.__on_failure_task.timeout or "60s",
                    inputs="{}",
                    parents=[],
                    retries=self.__on_failure_task.retries,
                    rate_limits=self.__on_failure_task.rate_limits,
                    backoff_factor=self.__on_failure_task.backoff_factor,
                    backoff_max_seconds=self.__on_failure_task.backoff_max_seconds,
                )
            ],
        )

    def _validate_priority(self, default_priority: int | None) -> int | None:
        validated_priority = (
            max(1, min(3, default_priority)) if default_priority else None
        )
        if validated_priority != default_priority:
            logger.warning(
                "Warning: Default Priority Must be between 1 and 3 -- inclusively. Adjusted to be within the range."
            )

        return validated_priority

    def get_create_opts(self, namespace: str) -> CreateWorkflowVersionOpts:
        service_name = self.get_service_name(namespace)

        name = self.get_name(namespace)
        event_triggers = [namespace + event for event in self.config.on_events]

        create_step_opts = [
            CreateWorkflowStepOpts(
                readable_id=task.name,
                action=service_name + ":" + task.name,
                timeout=task.timeout or "60s",
                inputs="{}",
                parents=[x.name for x in task.parents],
                retries=task.retries,
                rate_limits=task.rate_limits,
                worker_labels=task.desired_worker_labels,
                backoff_factor=task.backoff_factor,
                backoff_max_seconds=task.backoff_max_seconds,
            )
            for task in self.tasks
            if task.type == StepType.DEFAULT
        ]

        concurrency = self._validate_concurrency_actions(service_name)
        on_failure_job = self._validate_on_failure_task(name, service_name)
        validated_priority = self._validate_priority(self.config.default_priority)

        return CreateWorkflowVersionOpts(
            name=name,
            kind=WorkflowKind.DAG,
            version=self.config.version,
            event_triggers=event_triggers,
            cron_triggers=self.config.on_crons,
            schedule_timeout=self.config.schedule_timeout,
            sticky=maybe_int_to_str(
                convert_python_enum_to_proto(self.config.sticky, StickyStrategyProto)
            ),
            jobs=[
                CreateWorkflowJobOpts(
                    name=name,
                    steps=create_step_opts,
                )
            ],
            on_failure_job=on_failure_job,
            concurrency=concurrency,
            default_priority=validated_priority,
        )

    def create_run_workflow_config(
        self,
        input: TWorkflowInput | None = None,
        key: str | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunTriggerConfig:
        return WorkflowRunTriggerConfig(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
            key=key,
        )

    def run(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        return self.client.admin.run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

    async def aio_run(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        return await self.client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> list[WorkflowRunRef]:
        return self.client.admin.run_workflows(
            workflows=workflows,
            options=options,
        )

    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> list[WorkflowRunRef]:
        return await self.client.admin.aio_run_workflows(
            workflows=workflows,
            options=options,
        )

    def get_workflow_input(self, ctx: Context) -> TWorkflowInput:
        return cast(
            TWorkflowInput,
            self.config.input_validator.model_validate(ctx.workflow_input),
        )

    def schedule(
        self,
        schedules: list[datetime | timestamp_pb2.Timestamp],
        input: TWorkflowInput,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return self.client.admin.schedule_workflow(
            name=self.config.name,
            schedules=schedules,
            input=input.model_dump(),
            options=options,
        )

    async def aio_schedule(
        self,
        schedules: list[datetime | timestamp_pb2.Timestamp],
        input: TWorkflowInput,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return await self.client.admin.aio_schedule_workflow(
            name=self.config.name,
            schedules=schedules,
            input=input.model_dump(),
            options=options,
        )

    def task(
        self,
        name: str = "",
        timeout: str = "60m",
        parents: list[Task[TWorkflowInput, Any]] = [],
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                fn=func,
                workflow=self,
                type=StepType.DEFAULT,
                name=name.lower() or str(func.__name__).lower(),
                timeout=timeout,
                parents=parents,
                retries=retries,
                rate_limits=[r for rate_limit in rate_limits if (r := rate_limit._req)],
                desired_worker_labels={
                    key: transform_desired_worker_label(d)
                    for key, d in desired_worker_labels.items()
                },
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
            )

            self.__default_tasks.append(task)

            return task

        return inner

    def on_failure_task(
        self,
        name: str = "",
        timeout: str = "60m",
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                fn=func,
                workflow=self,
                type=StepType.ON_FAILURE,
                name=name.lower() or str(func.__name__).lower(),
                timeout=timeout,
                retries=retries,
                rate_limits=[r for rate_limit in rate_limits if (r := rate_limit._req)],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
            )

            self.__on_failure_task = task

            return task

        return inner
