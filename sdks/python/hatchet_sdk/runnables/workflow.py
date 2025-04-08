import asyncio
from datetime import datetime
from typing import TYPE_CHECKING, Any, Callable, Generic, Union, cast, overload

from google.protobuf import timestamp_pb2
from pydantic import BaseModel

from hatchet_sdk.clients.admin import (
    ScheduleTriggerWorkflowOptions,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.shared.condition_pb2 import TaskConditions
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    Concurrency,
    CreateTaskOpts,
    CreateWorkflowVersionRequest,
    DesiredWorkerLabels,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import StickyStrategy as StickyStrategyProto
from hatchet_sdk.contracts.workflows_pb2 import WorkflowVersion
from hatchet_sdk.labels import DesiredWorkerLabel
from hatchet_sdk.logger import logger
from hatchet_sdk.rate_limit import RateLimit
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import (
    DEFAULT_EXECUTION_TIMEOUT,
    DEFAULT_SCHEDULE_TIMEOUT,
    ConcurrencyExpression,
    EmptyModel,
    R,
    StepType,
    TWorkflowInput,
    WorkflowConfig,
)
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto
from hatchet_sdk.utils.timedelta_to_expression import Duration, timedelta_to_expr
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.waits import (
    Action,
    Condition,
    OrGroup,
    ParentCondition,
    SleepCondition,
    UserEventCondition,
)
from hatchet_sdk.workflow_run import WorkflowRunRef

if TYPE_CHECKING:
    from hatchet_sdk import Hatchet
    from hatchet_sdk.runnables.standalone import Standalone


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


class BaseWorkflow(Generic[TWorkflowInput]):
    def __init__(self, config: WorkflowConfig, client: "Hatchet") -> None:
        self.config = config
        self._default_tasks: list[Task[TWorkflowInput, Any]] = []
        self._durable_tasks: list[Task[TWorkflowInput, Any]] = []
        self._on_failure_task: Task[TWorkflowInput, Any] | None = None
        self._on_success_task: Task[TWorkflowInput, Any] | None = None
        self.client = client

    def _get_service_name(self, namespace: str) -> str:
        return f"{namespace}{self.config.name.lower()}"

    def _create_action_name(
        self, namespace: str, step: Task[TWorkflowInput, Any]
    ) -> str:
        return self._get_service_name(namespace) + ":" + step.name

    def _get_name(self, namespace: str) -> str:
        return namespace + self.config.name

    def _raise_for_invalid_concurrency(
        self, concurrency: ConcurrencyExpression
    ) -> bool:
        expr = concurrency.expression

        if not expr.startswith("input."):
            return True

        _, field = expr.split(".", maxsplit=2)

        if field not in self.config.input_validator.model_fields.keys():
            raise ValueError(
                f"The concurrency expression provided relies on the `{field}` field, which was not present in `{self.config.input_validator.__name__}`."
            )

        return True

    @overload
    def _concurrency_to_proto(self, concurrency: None) -> None: ...

    @overload
    def _concurrency_to_proto(
        self, concurrency: ConcurrencyExpression
    ) -> Concurrency: ...

    def _concurrency_to_proto(
        self, concurrency: ConcurrencyExpression | None
    ) -> Concurrency | None:
        if not concurrency:
            return None

        self._raise_for_invalid_concurrency(concurrency)

        return Concurrency(
            expression=concurrency.expression,
            max_runs=concurrency.max_runs,
            limit_strategy=concurrency.limit_strategy,
        )

    @overload
    def _validate_task(
        self, task: "Task[TWorkflowInput, R]", service_name: str
    ) -> CreateTaskOpts: ...

    @overload
    def _validate_task(self, task: None, service_name: str) -> None: ...

    def _validate_task(
        self, task: Union["Task[TWorkflowInput, R]", None], service_name: str
    ) -> CreateTaskOpts | None:
        if not task:
            return None

        return CreateTaskOpts(
            readable_id=task.name,
            action=service_name + ":" + task.name,
            timeout=timedelta_to_expr(task.execution_timeout),
            inputs="{}",
            parents=[p.name for p in task.parents],
            retries=task.retries,
            rate_limits=task.rate_limits,
            worker_labels=task.desired_worker_labels,
            backoff_factor=task.backoff_factor,
            backoff_max_seconds=task.backoff_max_seconds,
            concurrency=[self._concurrency_to_proto(t) for t in task.concurrency],
            conditions=self._conditions_to_proto(task),
            schedule_timeout=timedelta_to_expr(task.schedule_timeout),
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

    def _assign_action(self, condition: Condition, action: Action) -> Condition:
        condition.base.action = action

        return condition

    def _conditions_to_proto(self, task: Task[TWorkflowInput, Any]) -> TaskConditions:
        wait_for_conditions = [
            self._assign_action(w, Action.QUEUE) for w in task.wait_for
        ]

        cancel_if_conditions = [
            self._assign_action(c, Action.CANCEL) for c in task.cancel_if
        ]
        skip_if_conditions = [self._assign_action(s, Action.SKIP) for s in task.skip_if]

        conditions = wait_for_conditions + cancel_if_conditions + skip_if_conditions

        if len({c.base.readable_data_key for c in conditions}) != len(
            [c.base.readable_data_key for c in conditions]
        ):
            raise ValueError("Conditions must have unique readable data keys.")

        user_events = [
            c.to_pb() for c in conditions if isinstance(c, UserEventCondition)
        ]
        parent_overrides = [
            c.to_pb() for c in conditions if isinstance(c, ParentCondition)
        ]
        sleep_conditions = [
            c.to_pb() for c in conditions if isinstance(c, SleepCondition)
        ]

        return TaskConditions(
            parent_override_conditions=parent_overrides,
            sleep_conditions=sleep_conditions,
            user_event_conditions=user_events,
        )

    def _is_leaf_task(self, task: Task[TWorkflowInput, Any]) -> bool:
        return not any(task in t.parents for t in self.tasks if task != t)

    def _get_create_opts(self, namespace: str) -> CreateWorkflowVersionRequest:
        service_name = self._get_service_name(namespace)

        name = self._get_name(namespace)
        event_triggers = [namespace + event for event in self.config.on_events]

        if self._on_success_task:
            self._on_success_task.parents = [
                task
                for task in self.tasks
                if task.type == StepType.DEFAULT and self._is_leaf_task(task)
            ]

        on_success_task = self._validate_task(self._on_success_task, service_name)

        tasks = [
            self._validate_task(task, service_name)
            for task in self.tasks
            if task.type == StepType.DEFAULT
        ]

        if on_success_task:
            tasks += [on_success_task]

        on_failure_task = self._validate_task(self._on_failure_task, service_name)

        return CreateWorkflowVersionRequest(
            name=name,
            description=self.config.description,
            version=self.config.version,
            event_triggers=event_triggers,
            cron_triggers=self.config.on_crons,
            tasks=tasks,
            concurrency=self._concurrency_to_proto(self.config.concurrency),
            ## TODO: Fix this
            cron_input=None,
            on_failure_task=on_failure_task,
            sticky=convert_python_enum_to_proto(self.config.sticky, StickyStrategyProto),  # type: ignore[arg-type]
        )

    def _get_workflow_input(self, ctx: Context) -> TWorkflowInput:
        return cast(
            TWorkflowInput,
            self.config.input_validator.model_validate(ctx.workflow_input),
        )

    @property
    def tasks(self) -> list[Task[TWorkflowInput, Any]]:
        tasks = self._default_tasks + self._durable_tasks

        if self._on_failure_task:
            tasks += [self._on_failure_task]

        if self._on_success_task:
            tasks += [self._on_success_task]

        return tasks

    @property
    def is_durable(self) -> bool:
        return any(task.is_durable for task in self.tasks)

    @property
    def name(self) -> str:
        return self._get_name(self.client.config.namespace)

    def create_bulk_run_item(
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


class Workflow(BaseWorkflow[TWorkflowInput]):
    """
    A Hatchet workflow, which allows you to define tasks to be run and perform actions on the workflow, such as
    running / spawning children and scheduling future runs.
    """

    def run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        return self.client._client.admin.run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

    def run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        ref = self.client._client.admin.run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

        return ref.result()

    async def aio_run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        return await self.client._client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        ref = await self.client._client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=input.model_dump() if input else {},
            options=options,
        )

        return await ref.aio_result()

    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[dict[str, Any]]:
        refs = self.client._client.admin.run_workflows(
            workflows=workflows,
        )

        return [ref.result() for ref in refs]

    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[dict[str, Any]]:
        refs = await self.client._client.admin.aio_run_workflows(
            workflows=workflows,
        )

        return await asyncio.gather(*[ref.aio_result() for ref in refs])

    def run_many_no_wait(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        return self.client._client.admin.run_workflows(
            workflows=workflows,
        )

    async def aio_run_many_no_wait(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        return await self.client._client.admin.aio_run_workflows(
            workflows=workflows,
        )

    def schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput | None = None,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return self.client._client.admin.schedule_workflow(
            name=self.config.name,
            schedules=cast(list[datetime | timestamp_pb2.Timestamp], [run_at]),
            input=input.model_dump() if input else {},
            options=options,
        )

    async def aio_schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput,
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        return await self.client._client.admin.aio_schedule_workflow(
            name=self.config.name,
            schedules=cast(list[datetime | timestamp_pb2.Timestamp], [run_at]),
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
        self,
        name: str | None,
        func: (
            Callable[[TWorkflowInput, Context], R]
            | Callable[[TWorkflowInput, DurableContext], R]
        ),
    ) -> str:
        non_null_name = name or func.__name__

        return non_null_name.lower()

    def task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        parents: list[Task[TWorkflowInput, Any]] = [],
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: list[ConcurrencyExpression] = [],
        wait_for: list[Condition | OrGroup] = [],
        skip_if: list[Condition | OrGroup] = [],
        cancel_if: list[Condition | OrGroup] = [],
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        """
        A decorator to transform a function into a Hatchet task that run as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta | str

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

        :param concurrency: A list of concurrency expressions for the task. Defaults to an empty list (no concurrency).
        :type concurrency: list[ConcurrencyExpression]

        :param wait_for: A list of conditions that must be met before the task can run. Defaults to an empty list (no conditions).
        :type wait_for: list[Condition | OrGroup]

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped. Defaults to an empty list (no conditions).
        :type skip_if: list[Condition | OrGroup]

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled. Defaults to an empty list (no conditions).
        :type cancel_if: list[Condition | OrGroup]

        :returns: A decorator which creates a `Task` object.
        :rtype: Callable[[Callable[[Type[BaseModel], Context], R]], Task[Type[BaseModel], R]]
        """

        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                _fn=func,
                is_durable=False,
                workflow=self,
                type=StepType.DEFAULT,
                name=self._parse_task_name(name, func),
                execution_timeout=execution_timeout,
                schedule_timeout=schedule_timeout,
                parents=parents,
                retries=retries,
                rate_limits=[r.to_proto() for r in rate_limits],
                desired_worker_labels={
                    key: transform_desired_worker_label(d)
                    for key, d in desired_worker_labels.items()
                },
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
                wait_for=wait_for,
                skip_if=skip_if,
                cancel_if=cancel_if,
            )

            self._default_tasks.append(task)

            return task

        return inner

    def durable_task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        parents: list[Task[TWorkflowInput, Any]] = [],
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: list[ConcurrencyExpression] = [],
        wait_for: list[Condition | OrGroup] = [],
        skip_if: list[Condition | OrGroup] = [],
        cancel_if: list[Condition | OrGroup] = [],
    ) -> Callable[
        [Callable[[TWorkflowInput, DurableContext], R]], Task[TWorkflowInput, R]
    ]:
        """
        A decorator to transform a function into a durable Hatchet task that run as part of a workflow.

        **IMPORTANT:** This decorator creates a _durable_ task, which works using Hatchet's durable execution capabilities. This is an advanced feature of Hatchet.

        See the Hatchet docs for more information on durable execution to decide if this is right for you.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta | str

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

        :param concurrency: A list of concurrency expressions for the task. Defaults to an empty list (no concurrency).
        :type concurrency: list[ConcurrencyExpression]

        :param wait_for: A list of conditions that must be met before the task can run. Defaults to an empty list (no conditions).
        :type wait_for: list[Condition | OrGroup]

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped. Defaults to an empty list (no conditions).
        :type skip_if: list[Condition | OrGroup]

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled. Defaults to an empty list (no conditions).
        :type cancel_if: list[Condition | OrGroup]

        :returns: A decorator which creates a `Task` object.
        :rtype: Callable[[Callable[[Type[BaseModel], Context], R]], Task[Type[BaseModel], R]]
        """

        def inner(
            func: Callable[[TWorkflowInput, DurableContext], R]
        ) -> Task[TWorkflowInput, R]:
            task = Task(
                _fn=func,
                is_durable=True,
                workflow=self,
                type=StepType.DEFAULT,
                name=self._parse_task_name(name, func),
                execution_timeout=execution_timeout,
                schedule_timeout=schedule_timeout,
                parents=parents,
                retries=retries,
                rate_limits=[r.to_proto() for r in rate_limits],
                desired_worker_labels={
                    key: transform_desired_worker_label(d)
                    for key, d in desired_worker_labels.items()
                },
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
                wait_for=wait_for,
                skip_if=skip_if,
                cancel_if=cancel_if,
            )

            self._durable_tasks.append(task)

            return task

        return inner

    def on_failure_task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: list[ConcurrencyExpression] = [],
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        """
        A decorator to transform a function into a Hatchet on-failure task that runs as the last step in a workflow that had at least one task fail.

        :param name: The name of the on-failure task. If not specified, defaults to the name of the function being wrapped by the `on_failure_task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the on-failure task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta | str

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
                is_durable=False,
                _fn=func,
                workflow=self,
                type=StepType.ON_FAILURE,
                name=self._parse_task_name(name, func) + "-on-failure",
                execution_timeout=execution_timeout,
                schedule_timeout=schedule_timeout,
                retries=retries,
                rate_limits=[r.to_proto() for r in rate_limits],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
            )

            if self._on_failure_task:
                raise ValueError("Only one on-failure task is allowed")

            self._on_failure_task = task

            return task

        return inner

    def on_success_task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: list[ConcurrencyExpression] = [],
    ) -> Callable[[Callable[[TWorkflowInput, Context], R]], Task[TWorkflowInput, R]]:
        """
        A decorator to transform a function into a Hatchet on-success task that runs as the last step in a workflow that had all upstream tasks succeed.

        :param name: The name of the on-success task. If not specified, defaults to the name of the function being wrapped by the `on_failure_task` decorator.
        :type name: str | None

        :param timeout: The execution timeout of the on-success task. Defaults to 60 minutes.
        :type timeout: datetime.timedelta | str

        :param retries: The number of times to retry the on-success task before failing. Default: `0`
        :type retries: int

        :param rate_limits: A list of rate limit configurations for the on-success task. Defaults to an empty list (no rate limits).
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
                is_durable=False,
                _fn=func,
                workflow=self,
                type=StepType.ON_SUCCESS,
                name=self._parse_task_name(name, func) + "-on-success",
                execution_timeout=execution_timeout,
                schedule_timeout=schedule_timeout,
                retries=retries,
                rate_limits=[r.to_proto() for r in rate_limits],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
                parents=[],
            )

            if self._on_failure_task:
                raise ValueError("Only one on-failure task is allowed")

            self._on_success_task = task

            return task

        return inner

    def add_task(self, task: "Standalone[TWorkflowInput, Any]") -> None:
        """
        Add a task to a workflow. Intended to be used with a previously existing task (a Standalone),
        such as one created with `@hatchet.task()`, which has been converted to a `Task` object using `to_task`.

        For example:

        ```python
        @hatchet.task()
        def my_task(input, ctx) -> None:
            pass

        wf = hatchet.workflow()

        wf.add_task(my_task.to_task())
        ```
        """
        _task = task._task

        match _task.type:
            case StepType.DEFAULT:
                self._default_tasks.append(_task)
            case StepType.ON_FAILURE:
                if self._on_failure_task:
                    raise ValueError("Only one on-failure task is allowed")

                self._on_failure_task = _task
            case StepType.ON_SUCCESS:
                if self._on_success_task:
                    raise ValueError("Only one on-success task is allowed")

                self._on_success_task = _task
            case _:
                raise ValueError("Invalid task type")
