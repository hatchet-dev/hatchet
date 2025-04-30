import asyncio
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
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.workflows_pb2 import (
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
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.utils.typing import JSONSerializableMapping
from hatchet_sdk.waits import Condition, OrGroup
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

    @property
    def service_name(self) -> str:
        return f"{self.client.config.namespace}{self.config.name.lower()}"

    def _create_action_name(self, step: Task[TWorkflowInput, Any]) -> str:
        return self.service_name + ":" + step.name

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

    def _validate_priority(self, default_priority: int | None) -> int | None:
        validated_priority = (
            max(1, min(3, default_priority)) if default_priority else None
        )
        if validated_priority != default_priority:
            logger.warning(
                "Warning: Default Priority Must be between 1 and 3 -- inclusively. Adjusted to be within the range."
            )

        return validated_priority

    def _is_leaf_task(self, task: Task[TWorkflowInput, Any]) -> bool:
        return not any(task in t.parents for t in self.tasks if task != t)

    def to_proto(self) -> CreateWorkflowVersionRequest:
        namespace = self.client.config.namespace
        service_name = self.service_name

        name = self.name
        event_triggers = [namespace + event for event in self.config.on_events]

        if self._on_success_task:
            self._on_success_task.parents = [
                task
                for task in self.tasks
                if task.type == StepType.DEFAULT and self._is_leaf_task(task)
            ]

        on_success_task = (
            t.to_proto(service_name) if (t := self._on_success_task) else None
        )

        tasks = [
            task.to_proto(service_name)
            for task in self.tasks
            if task.type == StepType.DEFAULT
        ]

        if on_success_task:
            tasks += [on_success_task]

        on_failure_task = (
            t.to_proto(service_name) if (t := self._on_failure_task) else None
        )

        if isinstance(self.config.concurrency, list):
            _concurrency_arr = [c.to_proto() for c in self.config.concurrency]
            _concurrency = None
        elif isinstance(self.config.concurrency, ConcurrencyExpression):
            _concurrency_arr = []
            _concurrency = self.config.concurrency.to_proto()
        else:
            _concurrency = None
            _concurrency_arr = []

        return CreateWorkflowVersionRequest(
            name=name,
            description=self.config.description,
            version=self.config.version,
            event_triggers=event_triggers,
            cron_triggers=self.config.on_crons,
            tasks=tasks,
            ## TODO: Fix this
            cron_input=None,
            on_failure_task=on_failure_task,
            sticky=convert_python_enum_to_proto(self.config.sticky, StickyStrategyProto),  # type: ignore[arg-type]
            concurrency=_concurrency,
            concurrency_arr=_concurrency_arr,
            default_priority=self.config.default_priority,
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
        return self.client.config.namespace + self.config.name

    def create_bulk_run_item(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        key: str | None = None,
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunTriggerConfig:
        """
        Create a bulk run item for the workflow. This is intended to be used in conjunction with the various `run_many` methods.

        :param input: The input data for the workflow.
        :param key: The key for the workflow run. This is used to identify the run in the bulk operation and for deduplication.
        :param options: Additional options for the workflow run.

        :returns: A `WorkflowRunTriggerConfig` object that can be used to trigger the workflow run, which you then pass into the `run_many` methods.
        """
        return WorkflowRunTriggerConfig(
            workflow_name=self.config.name,
            input=self._serialize_input(input),
            options=options,
            key=key,
        )

    def _serialize_input(self, input: TWorkflowInput | None) -> JSONSerializableMapping:
        if not input:
            return {}

        if isinstance(input, BaseModel):
            return input.model_dump(mode="json")

        raise ValueError(
            f"Input must be a BaseModel or `None`, got {type(input)} instead."
        )


class Workflow(BaseWorkflow[TWorkflowInput]):
    """
    A Hatchet workflow, which allows you to define tasks to be run and perform actions on the workflow.

    Workflows in Hatchet represent coordinated units of work that can be triggered, scheduled, or run on a cron schedule.
    Each workflow can contain multiple tasks that can be arranged in dependencies (DAGs), have customized retry behavior,
    timeouts, concurrency controls, and more.

    Example:
    ```python
    from pydantic import BaseModel
    from hatchet_sdk import Hatchet

    class MyInput(BaseModel):
        name: str

    hatchet = Hatchet()
    workflow = hatchet.workflow("my-workflow", input_type=MyInput)

    @workflow.task()
    def greet(input, ctx):
        return f"Hello, {input.name}!"

    # Run the workflow
    result = workflow.run(MyInput(name="World"))
    ```

    Workflows support various execution patterns including:
    - One-time execution with `run()` or `aio_run()`
    - Scheduled execution with `schedule()`
    - Cron-based recurring execution with `create_cron()`
    - Bulk operations with `run_many()`

    Tasks within workflows can be defined with `@workflow.task()` or `@workflow.durable_task()` decorators
    and can be arranged into complex dependency patterns.
    """

    def run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        """
        Synchronously trigger a workflow run without waiting for it to complete.
        This method is useful for starting a workflow run and immediately returning a reference to the run without blocking while the workflow runs.

        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.

        :returns: A `WorkflowRunRef` object representing the reference to the workflow run.
        """
        return self.client._client.admin.run_workflow(
            workflow_name=self.config.name,
            input=self._serialize_input(input),
            options=options,
        )

    def run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        """
        Run the workflow synchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param options: Additional options for workflow execution like metadata and parent workflow ID.

        :returns: The result of the workflow execution as a dictionary.
        """

        ref = self.client._client.admin.run_workflow(
            workflow_name=self.config.name,
            input=self._serialize_input(input),
            options=options,
        )

        return ref.result()

    async def aio_run_no_wait(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> WorkflowRunRef:
        """
        Asynchronously trigger a workflow run without waiting for it to complete.
        This method is useful for starting a workflow run and immediately returning a reference to the run without blocking while the workflow runs.

        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.

        :returns: A `WorkflowRunRef` object representing the reference to the workflow run.
        """

        return await self.client._client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=self._serialize_input(input),
            options=options,
        )

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: TriggerWorkflowOptions = TriggerWorkflowOptions(),
    ) -> dict[str, Any]:
        """
        Run the workflow asynchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param options: Additional options for workflow execution like metadata and parent workflow ID.

        :returns: The result of the workflow execution as a dictionary.
        """
        ref = await self.client._client.admin.aio_run_workflow(
            workflow_name=self.config.name,
            input=self._serialize_input(input),
            options=options,
        )

        return await ref.aio_result()

    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[dict[str, Any]]:
        """
        Run a workflow in bulk and wait for all runs to complete.
        This method triggers multiple workflow runs, blocks until all of them complete, and returns the final results.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of results for each workflow run.
        """
        refs = self.client._client.admin.run_workflows(
            workflows=workflows,
        )

        return [ref.result() for ref in refs]

    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[dict[str, Any]]:
        """
        Run a workflow in bulk and wait for all runs to complete.
        This method triggers multiple workflow runs, blocks until all of them complete, and returns the final results.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of results for each workflow run.
        """
        refs = await self.client._client.admin.aio_run_workflows(
            workflows=workflows,
        )

        return await asyncio.gather(*[ref.aio_result() for ref in refs])

    def run_many_no_wait(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        """
        Run a workflow in bulk without waiting for all runs to complete.

        This method triggers multiple workflow runs and immediately returns a list of references to the runs without blocking while the workflows run.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :returns: A list of `WorkflowRunRef` objects, each representing a reference to a workflow run.
        """
        return self.client._client.admin.run_workflows(
            workflows=workflows,
        )

    async def aio_run_many_no_wait(
        self,
        workflows: list[WorkflowRunTriggerConfig],
    ) -> list[WorkflowRunRef]:
        """
        Run a workflow in bulk without waiting for all runs to complete.

        This method triggers multiple workflow runs and immediately returns a list of references to the runs without blocking while the workflows run.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.

        :returns: A list of `WorkflowRunRef` objects, each representing a reference to a workflow run.
        """
        return await self.client._client.admin.aio_run_workflows(
            workflows=workflows,
        )

    def schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        """
        Schedule a workflow to run at a specific time.

        :param run_at: The time at which to schedule the workflow.
        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.
        :returns: A `WorkflowVersion` object representing the scheduled workflow.
        """
        return self.client._client.admin.schedule_workflow(
            name=self.config.name,
            schedules=cast(list[datetime | timestamp_pb2.Timestamp], [run_at]),
            input=self._serialize_input(input),
            options=options,
        )

    async def aio_schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        options: ScheduleTriggerWorkflowOptions = ScheduleTriggerWorkflowOptions(),
    ) -> WorkflowVersion:
        """
        Schedule a workflow to run at a specific time.

        :param run_at: The time at which to schedule the workflow.
        :param input: The input data for the workflow.
        :param options: Additional options for workflow execution.
        :returns: A `WorkflowVersion` object representing the scheduled workflow.
        """
        return await self.client._client.admin.aio_schedule_workflow(
            name=self.config.name,
            schedules=cast(list[datetime | timestamp_pb2.Timestamp], [run_at]),
            input=self._serialize_input(input),
            options=options,
        )

    def create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        additional_metadata: JSONSerializableMapping = {},
        priority: int | None = None,
    ) -> CronWorkflows:
        """
        Create a cron job for the workflow.

        :param cron_name: The name of the cron job.
        :param expression: The cron expression that defines the schedule for the cron job.
        :param input: The input data for the workflow.
        :param additional_metadata: Additional metadata for the cron job.
        :param priority: The priority of the cron job. Must be between 1 and 3, inclusive.

        :returns: A `CronWorkflows` object representing the created cron job.
        """
        return self.client.cron.create(
            workflow_name=self.config.name,
            cron_name=cron_name,
            expression=expression,
            input=self._serialize_input(input),
            additional_metadata=additional_metadata,
            priority=priority,
        )

    async def aio_create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        additional_metadata: JSONSerializableMapping = {},
        priority: int | None = None,
    ) -> CronWorkflows:
        """
        Create a cron job for the workflow.

        :param cron_name: The name of the cron job.
        :param expression: The cron expression that defines the schedule for the cron job.
        :param input: The input data for the workflow.
        :param additional_metadata: Additional metadata for the cron job.
        :param priority: The priority of the cron job. Must be between 1 and 3, inclusive.

        :returns: A `CronWorkflows` object representing the created cron job.
        """
        return await self.client.cron.aio_create(
            workflow_name=self.config.name,
            cron_name=cron_name,
            expression=expression,
            input=self._serialize_input(input),
            additional_metadata=additional_metadata,
            priority=priority,
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
        A decorator to transform a function into a Hatchet task that runs as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param parents: A list of tasks that are parents of the task. Note: Parents must be defined before their children.

        :param retries: The number of times to retry the task before failing.

        :param rate_limits: A list of rate limit configurations for the task.

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned. See documentation and examples on affinity and worker labels for more details.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the task.

        :param wait_for: A list of conditions that must be met before the task can run.

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped.

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled.

        :returns: A decorator which creates a `Task` object.
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
        A decorator to transform a function into a durable Hatchet task that runs as part of a workflow.

        **IMPORTANT:** This decorator creates a _durable_ task, which works using Hatchet's durable execution capabilities. This is an advanced feature of Hatchet.

        See the Hatchet docs for more information on durable execution to decide if this is right for you.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param parents: A list of tasks that are parents of the task. Note: Parents must be defined before their children.

        :param retries: The number of times to retry the task before failing.

        :param rate_limits: A list of rate limit configurations for the task.

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned. See documentation and examples on affinity and worker labels for more details.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the task.

        :param wait_for: A list of conditions that must be met before the task can run.

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped.

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled.

        :returns: A decorator which creates a `Task` object.
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

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param retries: The number of times to retry the on-failure task before failing.

        :param rate_limits: A list of rate limit configurations for the on-failure task.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the on-success task.

        :returns: A decorator which creates a `Task` object.
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

        :param name: The name of the on-success task. If not specified, defaults to the name of the function being wrapped by the `on_success_task` decorator.

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param retries: The number of times to retry the on-success task before failing

        :param rate_limits: A list of rate limit configurations for the on-success task.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the on-success task.

        :returns: A decorator which creates a Task object.
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

            if self._on_success_task:
                raise ValueError("Only one on-success task is allowed")

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

    def list_runs(
        self,
        since: datetime | None = None,
        until: datetime | None = None,
        limit: int = 100,
        offset: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        only_tasks: bool = False,
    ) -> list[V1TaskSummary]:
        """
        List runs of the workflow.

        :param since: The start time for the runs to be listed.
        :param until: The end time for the runs to be listed.
        :param limit: The maximum number of runs to be listed.
        :param offset: The offset for pagination.
        :param statuses: The statuses of the runs to be listed.
        :param additional_metadata: Additional metadata for filtering the runs.
        :param worker_id: The ID of the worker that ran the tasks.
        :param parent_task_external_id: The external ID of the parent task.
        :param only_tasks: Whether to list only task runs.

        :returns: A list of `V1TaskSummary` objects representing the runs of the workflow.
        """
        workflows = self.client.workflows.list(workflow_name=self.name)

        if not workflows.rows:
            logger.warning(f"No runs found for {self.name}")
            return []

        workflow = workflows.rows[0]

        response = self.client.runs.list(
            workflow_ids=[workflow.metadata.id],
            since=since or datetime.now() - timedelta(days=1),
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
        )

        return response.rows

    async def aio_list_runs(
        self,
        since: datetime | None = None,
        until: datetime | None = None,
        limit: int = 100,
        offset: int | None = None,
        statuses: list[V1TaskStatus] | None = None,
        additional_metadata: dict[str, str] | None = None,
        worker_id: str | None = None,
        parent_task_external_id: str | None = None,
        only_tasks: bool = False,
    ) -> list[V1TaskSummary]:
        """
        List runs of the workflow.

        :param since: The start time for the runs to be listed.
        :param until: The end time for the runs to be listed.
        :param limit: The maximum number of runs to be listed.
        :param offset: The offset for pagination.
        :param statuses: The statuses of the runs to be listed.
        :param additional_metadata: Additional metadata for filtering the runs.
        :param worker_id: The ID of the worker that ran the tasks.
        :param parent_task_external_id: The external ID of the parent task.
        :param only_tasks: Whether to list only task runs.

        :returns: A list of `V1TaskSummary` objects representing the runs of the workflow.
        """
        return await asyncio.to_thread(
            self.list_runs,
            since=since or datetime.now() - timedelta(days=1),
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
        )
