from datetime import datetime, timedelta
from typing import TYPE_CHECKING, Any, Callable, Generic, cast

from google.protobuf import timestamp_pb2
from pydantic import BaseModel

from hatchet_sdk.clients.admin import (
    ScheduleTriggerWorkflowOptions,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.context.context import Context
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
from hatchet_sdk.utils.timedelta_to_expression import timedelta_to_expr
from hatchet_sdk.utils.typing import JSONSerializableMapping
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
    A Hatchet workflow, which allows you to define tasks to be run and perform actions on the workflow, such as
    running / spawning children and scheduling future runs.
    """

    def __init__(self, config: WorkflowConfig, client: "Hatchet") -> None:
        self.config = config
        self._default_tasks: list[Task[TWorkflowInput, Any]] = []
        self._on_failure_task: Task[TWorkflowInput, Any] | None = None
        self.client = client

    def _get_service_name(self, namespace: str) -> str:
        return f"{namespace}{self.config.name.lower()}"

    def _create_action_name(
        self, namespace: str, step: Task[TWorkflowInput, Any]
    ) -> str:
        return self._get_service_name(namespace) + ":" + step.name

    def _get_name(self, namespace: str) -> str:
        return namespace + self.config.name

    def _validate_concurrency_options(self) -> WorkflowConcurrencyOpts | None:
        if not self.config.concurrency:
            return None

        return WorkflowConcurrencyOpts(
            expression=self.config.concurrency.expression,
            max_runs=self.config.concurrency.max_runs,
            limit_strategy=self.config.concurrency.limit_strategy,
        )

    def _validate_on_failure_task(
        self, name: str, service_name: str
    ) -> CreateWorkflowJobOpts | None:
        if not self._on_failure_task:
            return None

        return CreateWorkflowJobOpts(
            name=name + "-on-failure",
            steps=[
                CreateWorkflowStepOpts(
                    readable_id=self._on_failure_task.name,
                    action=service_name + ":" + self._on_failure_task.name,
                    timeout=timedelta_to_expr(self._on_failure_task.timeout) or "60s",
                    inputs="{}",
                    parents=[],
                    retries=self._on_failure_task.retries,
                    rate_limits=self._on_failure_task.rate_limits,
                    backoff_factor=self._on_failure_task.backoff_factor,
                    backoff_max_seconds=self._on_failure_task.backoff_max_seconds,
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

    def _get_create_opts(self, namespace: str) -> CreateWorkflowVersionOpts:
        service_name = self._get_service_name(namespace)

        name = self._get_name(namespace)
        event_triggers = [namespace + event for event in self.config.on_events]

        create_step_opts = [
            CreateWorkflowStepOpts(
                readable_id=task.name,
                action=service_name + ":" + task.name,
                timeout=timedelta_to_expr(task.timeout) or "60s",
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

        on_failure_job = self._validate_on_failure_task(name, service_name)

        return CreateWorkflowVersionOpts(
            name=name,
            kind=WorkflowKind.DAG,
            version=self.config.version,
            event_triggers=event_triggers,
            cron_triggers=self.config.on_crons,
            schedule_timeout=timedelta_to_expr(self.config.schedule_timeout),
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
            concurrency=self._validate_concurrency_options(),
            default_priority=self.config.default_priority,
        )

    def _get_workflow_input(self, ctx: Context) -> TWorkflowInput:
        return cast(
            TWorkflowInput,
            self.config.input_validator.model_validate(ctx.workflow_input),
        )

    @property
    def tasks(self) -> list[Task[TWorkflowInput, Any]]:
        tasks = self._default_tasks

        if self._on_failure_task:
            tasks += [self._on_failure_task]

        return tasks

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

    def run_and_get_result(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        ref = self.client.admin.run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

        return ref.result()

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

    async def aio_run_and_get_result(
        self,
        input: TWorkflowInput | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        ref = await self.client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

        return await ref.aio_result()

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

    def create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        return self.client.cron.create(
            workflow_name=self.config.name,
            cron_name=cron_name,
            expression=expression,
            input=input.model_dump(),
            additional_metadata=additional_metadata,
        )

    async def aio_create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput,
        additional_metadata: JSONSerializableMapping,
    ) -> CronWorkflows:
        return await self.client.cron.aio_create(
            workflow_name=self.config.name,
            cron_name=cron_name,
            expression=expression,
            input=input.model_dump(),
            additional_metadata=additional_metadata,
        )

    def _parse_task_name(
        self, name: str | None, func: Callable[[TWorkflowInput, Context], R]
    ) -> str:
        non_null_name = name or func.__name__

        return non_null_name.lower()

    def task(
        self,
        name: str | None = None,
        timeout: timedelta = timedelta(minutes=60),
        parents: list[Task[TWorkflowInput, Any]] = [],
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        """
        A decorator to transform a function into a Hatchet task that run as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta

        :param parents: A list of tasks that are parents of the task. Note: Parents must be defined before their children. Defaults to an empty list (no parents).
        :type parents: list[Task]

        :param retries: The number of times to retry the task before failing. Default: `0`
        :type retries: int

        :param rate_limits: A list of rate limit configurations for the task. Defaults to an empty list (no rate limits).
        :type rate_limits: list[RateLimit]

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned. See documentation and examples on affinity and worker labels for more details. Defaults to an empty dictionary (no desired worker labels).
        :type desired_worker_labels: dict[str, DesiredWorkerLabel]

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries. Default: `None`
        :type backoff_factor: float | None

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue. Default: `None`
        :type backoff_max_seconds: int | None

        :returns: A decorator which creates a `Task` object.
        :rtype: Callable[[Callable[[Type[BaseModel], Context], R]], Task[Type[BaseModel], R]]
        """

        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                fn=func,
                workflow=self,
                type=StepType.DEFAULT,
                name=self._parse_task_name(name, func),
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

            self._default_tasks.append(task)

            return task

        return inner

    def on_failure_task(
        self,
        name: str | None = None,
        timeout: timedelta = timedelta(minutes=60),
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        """
        A decorator to transform a function into a Hatchet on-failure task that runs as the last step in a workflow that had at least one task fail.

        :param name: The name of the on-failure task. If not specified, defaults to the name of the function being wrapped by the `on_failure_task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the on-failure task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta

        :param retries: The number of times to retry the on-failure task before failing. Default: `0`
        :type retries: int

        :param rate_limits: A list of rate limit configurations for the on-failure task. Defaults to an empty list (no rate limits).
        :type rate_limits: list[RateLimit]

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries. Default: `None`
        :type backoff_factor: float | None

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue. Default: `None`
        :type backoff_max_seconds: int | None

        :returns: A decorator which creates a `Task` object.
        :rtype: Callable[[Callable[[Type[BaseModel], Context], R]], Task[Type[BaseModel], R]]
        """

        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                fn=func,
                workflow=self,
                type=StepType.ON_FAILURE,
                name=self._parse_task_name(name, func),
                timeout=timeout,
                retries=retries,
                rate_limits=[r for rate_limit in rate_limits if (r := rate_limit._req)],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
            )

            self._on_failure_task = task

            return task

        return inner
