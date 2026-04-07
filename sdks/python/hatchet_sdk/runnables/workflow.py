import asyncio
import json
from collections.abc import Callable
from datetime import datetime, timedelta
from enum import Enum
from functools import cached_property
from typing import (
    TYPE_CHECKING,
    Any,
    Concatenate,
    Generic,
    Literal,
    ParamSpec,
    TypeVar,
    cast,
    get_type_hints,
    overload,
)

from pydantic import BaseModel, ConfigDict, SkipValidation, TypeAdapter, model_validator
from typing_extensions import assert_never

from hatchet_sdk.clients.listeners.run_event_listener import RunEventListener
from hatchet_sdk.clients.rest.models.cron_workflows import CronWorkflows
from hatchet_sdk.clients.rest.models.v1_filter import V1Filter
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary
from hatchet_sdk.conditions import Condition, OrGroup
from hatchet_sdk.context.context import Context, DurableContext
from hatchet_sdk.contracts.v1.workflows_pb2 import (
    CreateWorkflowVersionRequest,
)
from hatchet_sdk.contracts.v1.workflows_pb2 import StickyStrategy as StickyStrategyProto
from hatchet_sdk.contracts.workflows_pb2 import WorkflowVersion
from hatchet_sdk.labels import DesiredWorkerLabel
from hatchet_sdk.rate_limit import RateLimit
from hatchet_sdk.runnables.contextvars import (
    ctx_durable_context,
)
from hatchet_sdk.runnables.eviction import (
    DEFAULT_DURABLE_TASK_EVICTION_POLICY,
    EvictionPolicy,
)
from hatchet_sdk.runnables.task import Task
from hatchet_sdk.runnables.types import (
    EmptyModel,
    R,
    StepType,
    TaskDefaults,
    TaskPayloadForInternalUse,
    TWorkflowInput,
    WorkflowConfig,
    normalize_validator,
)
from hatchet_sdk.serde import HATCHET_PYDANTIC_SENTINEL
from hatchet_sdk.types.concurrency import ConcurrencyExpression
from hatchet_sdk.types.priority import Priority
from hatchet_sdk.types.trigger import (
    ScheduleTriggerWorkflowOptions,
    TriggerWorkflowOptions,
    WorkflowRunTriggerConfig,
)
from hatchet_sdk.utils.aio import gather_max_concurrency
from hatchet_sdk.utils.proto_enums import convert_python_enum_to_proto
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.utils.typing import CoroutineLike, JSONSerializableMapping
from hatchet_sdk.workflow_run import WorkflowRunRef

if TYPE_CHECKING:
    from agents import FunctionTool
    from claude_agent_sdk import SdkMcpTool

    from hatchet_sdk import Hatchet


T = TypeVar("T")
P = ParamSpec("P")


# Once support for 3.10 is dropped, convert this to StrEnum
class MCPProvider(str, Enum):
    CLAUDE = "CLAUDE"
    OPENAI = "OPENAI"


def fall_back_to_default(value: T, param_default: T, fallback_value: T | None) -> T:
    ## If the value is not the param default, it's set
    if value != param_default:
        return value

    ## Otherwise, it's unset, so return the fallback value if it's set
    if fallback_value is not None:
        return fallback_value

    ## Otherwise return the param value
    return value


class ComputedTaskParameters(BaseModel):
    schedule_timeout: Duration
    execution_timeout: Duration
    retries: int
    backoff_factor: float | None
    backoff_max_seconds: int | None

    task_defaults: TaskDefaults

    @model_validator(mode="after")
    def validate_params(self) -> "ComputedTaskParameters":
        self.execution_timeout = fall_back_to_default(
            value=self.execution_timeout,
            param_default=timedelta(seconds=60),
            fallback_value=self.task_defaults.execution_timeout,
        )
        self.schedule_timeout = fall_back_to_default(
            value=self.schedule_timeout,
            param_default=timedelta(minutes=5),
            fallback_value=self.task_defaults.schedule_timeout,
        )
        self.backoff_factor = fall_back_to_default(
            value=self.backoff_factor,
            param_default=None,
            fallback_value=self.task_defaults.backoff_factor,
        )
        self.backoff_max_seconds = fall_back_to_default(
            value=self.backoff_max_seconds,
            param_default=None,
            fallback_value=self.task_defaults.backoff_max_seconds,
        )
        self.retries = fall_back_to_default(
            value=self.retries,
            param_default=0,
            fallback_value=self.task_defaults.retries,
        )

        return self


class TypedTriggerWorkflowRunConfig(BaseModel, Generic[TWorkflowInput]):
    model_config = ConfigDict(arbitrary_types_allowed=True)
    input: SkipValidation[TWorkflowInput]
    options: TriggerWorkflowOptions


class BaseWorkflow(Generic[TWorkflowInput]):
    def __init__(self, config: WorkflowConfig, client: "Hatchet") -> None:
        self._config = config
        self._default_tasks: list[Task[TWorkflowInput, Any]] = []
        self._durable_tasks: list[Task[TWorkflowInput, Any]] = []
        self._on_failure_task: Task[TWorkflowInput, Any] | None = None
        self._on_success_task: Task[TWorkflowInput, Any] | None = None
        self._client = client

    @property
    def service_name(self) -> str:
        return self._client.config.apply_namespace(self._config.name.lower())

    def _create_action_name(self, step: Task[TWorkflowInput, Any]) -> str:
        return self.service_name + ":" + step.name

    def _is_leaf_task(self, task: Task[TWorkflowInput, Any]) -> bool:
        return not any(task in t.parents for t in self.tasks if task != t)

    def to_proto(self) -> CreateWorkflowVersionRequest:
        namespace = self._client.config.namespace
        service_name = self.service_name

        name = self.name
        event_triggers = [
            self._client.config.apply_namespace(event, namespace)
            for event in self._config.on_events
        ]

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

        if isinstance(self._config.concurrency, list):
            _concurrency_arr = [c.to_proto() for c in self._config.concurrency]
            _concurrency = None
        elif isinstance(self._config.concurrency, ConcurrencyExpression):
            _concurrency_arr = []
            _concurrency = self._config.concurrency.to_proto()
        elif isinstance(self._config.concurrency, int):
            _concurrency_arr = []
            _concurrency = ConcurrencyExpression.from_int(
                self._config.concurrency
            ).to_proto()
        else:
            _concurrency = None
            _concurrency_arr = []

        # Hack to not send a JSON schema if the input type is None/EmptyModel
        input_type = self._config.input_validator.core_schema.get("cls")

        if input_type is None or input_type is EmptyModel:
            json_schema = None
        else:
            try:
                json_schema = json.dumps(
                    self._config.input_validator.json_schema()
                ).encode("utf-8")
            except Exception:
                json_schema = None

        return CreateWorkflowVersionRequest(
            name=name,
            description=self._config.description,
            version=self._config.version,
            event_triggers=event_triggers,
            cron_triggers=self._config.on_crons,
            tasks=tasks,
            ## TODO: Fix this
            cron_input=None,
            on_failure_task=on_failure_task,
            sticky=convert_python_enum_to_proto(
                self._config.sticky, StickyStrategyProto
            ),  # type: ignore[arg-type]
            concurrency=_concurrency,
            concurrency_arr=_concurrency_arr,
            default_priority=self._config.default_priority,
            default_filters=[f.to_proto() for f in self._config.default_filters],
            input_json_schema=json_schema,
        )

    def _get_workflow_input(self, ctx: Context) -> TWorkflowInput:
        return cast(
            TWorkflowInput,
            self._config.input_validator.validate_python(
                ctx._workflow_input, context=HATCHET_PYDANTIC_SENTINEL
            ),
        )

    def _combine_additional_metadata(
        self, additional_metadata_from_trigger: JSONSerializableMapping
    ) -> JSONSerializableMapping:
        return {
            **self._config.default_additional_metadata,
            **additional_metadata_from_trigger,
        }

    def _create_trigger_run_options_with_combined_additional_meta(
        self,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> TriggerWorkflowOptions:
        return TriggerWorkflowOptions(
            child_key=child_key,
            additional_metadata=self._combine_additional_metadata(
                additional_metadata or {}
            ),
            priority=priority,
            sticky=sticky,
            desired_worker_id=desired_worker_id,
            desired_worker_label=desired_worker_labels,
        )

    def _create_schedule_options_with_combined_metadata(
        self,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> ScheduleTriggerWorkflowOptions:
        return ScheduleTriggerWorkflowOptions(
            child_key=child_key,
            additional_metadata=self._combine_additional_metadata(
                additional_metadata or {}
            ),
            priority=priority,
        )

    @property
    def input_validator(self) -> TypeAdapter[TWorkflowInput]:
        return cast(TypeAdapter[TWorkflowInput], self._config.input_validator)

    @property
    def input_validator_type(self) -> type[TWorkflowInput]:
        return cast(type[TWorkflowInput], self._config.input_validator._type)

    @property
    def tasks(self) -> list[Task[TWorkflowInput, Any]]:
        tasks = self._default_tasks + self._durable_tasks

        if self._on_failure_task:
            tasks += [self._on_failure_task]

        if self._on_success_task:
            tasks += [self._on_success_task]

        return tasks

    @property
    def name(self) -> str:
        """
        The (namespaced) name of the workflow.
        """
        return self._client.config.namespace + self._config.name

    def create_bulk_run_item(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        key: str | None = None,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: Priority | None = None,
        desired_worker_id: str | None = None,
        sticky: bool = False,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> WorkflowRunTriggerConfig:
        """
        Create a bulk run item for the workflow. This is intended to be used in conjunction with the various `run_many` methods.

        :param input: The input data for the workflow.
        :param key: The key for the workflow run. This is used to identify the run in the bulk operation and for deduplication.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the workflow run.
        :param desired_worker_id: The ID of the desired worker to run the workflow on.
        :param sticky: Whether to use sticky scheduling for the workflow run.
        :param desired_worker_labels: A list of desired worker labels for worker affinity.

        :returns: A `WorkflowRunTriggerConfig` object that can be used to trigger the workflow run, which you then pass into the `run_many` methods.
        """
        return WorkflowRunTriggerConfig(
            workflow_name=self._config.name,
            input=self._serialize_input(input, target="string"),
            options=self._create_trigger_run_options_with_combined_additional_meta(
                child_key=child_key,
                additional_metadata=additional_metadata,
                priority=priority,
                sticky=sticky,
                desired_worker_id=desired_worker_id,
                desired_worker_labels=desired_worker_labels,
            ),
            key=key,
        )

    def _serialize_input_to_str(self, input: TWorkflowInput | None) -> str | None:
        return self._config.input_validator.dump_json(
            input,  # type: ignore[arg-type]
            context=HATCHET_PYDANTIC_SENTINEL,
        ).decode("utf-8")

    def _serialize_input_to_dict(
        self, input: TWorkflowInput | None
    ) -> JSONSerializableMapping:
        return cast(
            JSONSerializableMapping,
            self._config.input_validator.dump_python(
                input,  # type: ignore[arg-type]
                mode="json",
                context=HATCHET_PYDANTIC_SENTINEL,
            ),
        )

    @overload
    def _serialize_input(
        self, input: TWorkflowInput | None, target: Literal["string"] = "string"
    ) -> str | None: ...

    @overload
    def _serialize_input(
        self, input: TWorkflowInput | None, target: Literal["dict"] = "dict"
    ) -> JSONSerializableMapping: ...

    def _serialize_input(
        self,
        input: TWorkflowInput | None,
        target: Literal["string"] | Literal["dict"] = "string",
    ) -> JSONSerializableMapping | str | None:
        if not input:
            return None

        if target == "string":
            return self._serialize_input_to_str(input)

        if target == "dict":
            return self._serialize_input_to_dict(input)

        raise ValueError(f"Invalid target for input serialization: {target}")

    @cached_property
    def id(self) -> str:
        """
        Get the ID of the workflow.

        :raises ValueError: If no workflow ID is found for the workflow name.
        :returns: The ID of the workflow.
        """
        workflows = self._client.workflows.list(workflow_name=self.name)

        if not workflows.rows:
            raise ValueError(f"No id found for {self.name}")

        for workflow in workflows.rows:
            if workflow.name == self.name:
                return workflow.metadata.id

        raise ValueError(f"No id found for {self.name}")

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
        triggering_event_external_id: str | None = None,
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
        :param triggering_event_external_id: The event id that triggered the task run.

        :returns: A list of `V1TaskSummary` objects representing the runs of the workflow.
        """
        return self._client.runs.list_with_pagination(
            workflow_ids=[self.id],
            since=since,
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
            triggering_event_external_id=triggering_event_external_id,
        )

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
        triggering_event_external_id: str | None = None,
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
        :param triggering_event_external_id: The event id that triggered the task run.

        :returns: A list of `V1TaskSummary` objects representing the runs of the workflow.
        """
        return await self._client.runs.aio_list_with_pagination(
            workflow_ids=[self.id],
            since=since,
            only_tasks=only_tasks,
            offset=offset,
            limit=limit,
            statuses=statuses,
            until=until,
            additional_metadata=additional_metadata,
            worker_id=worker_id,
            parent_task_external_id=parent_task_external_id,
            triggering_event_external_id=triggering_event_external_id,
        )

    def create_filter(
        self,
        expression: str,
        scope: str,
        payload: JSONSerializableMapping | None = None,
    ) -> V1Filter:
        """
        Create a new filter.

        :param expression: The expression to evaluate for the filter.
        :param scope: The scope for the filter.
        :param payload: The payload to send with the filter.

        :return: The created filter.
        """
        return self._client.filters.create(
            workflow_id=self.id,
            expression=expression,
            scope=scope,
            payload=payload,
        )

    async def aio_create_filter(
        self,
        expression: str,
        scope: str,
        payload: JSONSerializableMapping | None = None,
    ) -> V1Filter:
        """
        Create a new filter.

        :param expression: The expression to evaluate for the filter.
        :param scope: The scope for the filter.
        :param payload: The payload to send with the filter.

        :return: The created filter.
        """
        return await asyncio.to_thread(
            self.create_filter,
            expression=expression,
            scope=scope,
            payload=payload,
        )

    def schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
    ) -> WorkflowVersion:
        """
        Schedule a workflow to run at a specific time.

        :param run_at: The time at which to schedule the workflow.
        :param input: The input data for the workflow.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the scheduled workflow run.

        :returns: A `WorkflowVersion` object representing the scheduled workflow.
        """
        opts = self._create_schedule_options_with_combined_metadata(
            child_key=child_key,
            additional_metadata=additional_metadata,
            priority=priority,
        )

        return self._client._client.admin.schedule_workflow(
            name=self._config.name,
            schedules=[run_at],
            input=self._serialize_input(input, target="string"),
            options=opts,
        )

    async def aio_schedule(
        self,
        run_at: datetime,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
    ) -> WorkflowVersion:
        """
        Schedule a workflow to run at a specific time.

        :param run_at: The time at which to schedule the workflow.
        :param input: The input data for the workflow.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the scheduled workflow run.

        :returns: A `WorkflowVersion` object representing the scheduled workflow.
        """
        return await asyncio.to_thread(
            self.schedule,
            run_at=run_at,
            input=input,
            child_key=child_key,
            additional_metadata=additional_metadata,
            priority=priority,
        )

    def create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | Priority | None = None,
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
        return self._client.cron.create(
            workflow_name=self._config.name,
            cron_name=cron_name,
            expression=expression,
            input=self._serialize_input(input, target="dict"),
            additional_metadata=additional_metadata or {},
            priority=priority,
        )

    async def aio_create_cron(
        self,
        cron_name: str,
        expression: str,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | Priority | None = None,
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
        return await asyncio.to_thread(
            self.create_cron,
            cron_name=cron_name,
            expression=expression,
            input=input,
            additional_metadata=additional_metadata,
            priority=priority,
        )

    def delete(self) -> None:
        """
        Permanently delete the workflow.

        **DANGEROUS: This will delete a workflow and all of its data**
        """
        self._client.workflows.delete(self.id)

    async def aio_delete(self) -> None:
        """
        Permanently delete the workflow.

        **DANGEROUS: This will delete a workflow and all of its data**
        """
        await self._client.workflows.aio_delete(self.id)

    @overload
    def mcp_tool(
        self,
        provider: Literal[MCPProvider.CLAUDE],
        **kwargs: Any,
    ) -> "SdkMcpTool[TWorkflowInput]": ...
    @overload
    def mcp_tool(
        self,
        provider: Literal[MCPProvider.OPENAI],
        **kwargs: Any,
    ) -> "FunctionTool": ...
    def mcp_tool(
        self,
        provider: MCPProvider,
        **kwargs: Any,
    ) -> "FunctionTool | SdkMcpTool[TWorkflowInput]":
        """
        Creates a wrapper around the workflow enabling its usage in MCP server implementations.
        Supports Claude and OpenAI agent SDKs, requires installing the `claude` or `openai` extra using (e.g.) `pip install hatchet-sdk[claude]`.

        :param provider: The Agent provider you are using the tool with.
        :param **kwargs: Additional arguments that will be passed to the underlying MCP Tool object constructor.


        :raises ValueError: if runnable does not have a description.
        :raises NotImplementedError: If provider does not exist.

        :returns: The MCP tool configuration object.
        """
        if not self._config.description:
            raise ValueError(
                f"Runnable '{self._config.name}' has no description. "
                "Set description= when defining the workflow or task."
            )
        description = self._config.description
        if self.input_validator_type is EmptyModel:
            raise ValueError(
                f"Runnable '{self._config.name}' has no input validator. "
                "Set input_validator= when defining the workflow or task."
            )
        input_schema = self.input_validator.json_schema()
        if isinstance(self, Workflow):
            match provider:
                case MCPProvider.CLAUDE:
                    from hatchet_sdk.runnables.mcp.claude import workflow_to_claude_mcp

                    return workflow_to_claude_mcp(
                        self, input_schema, description, **kwargs
                    )
                case MCPProvider.OPENAI:
                    from hatchet_sdk.runnables.mcp.openai import workflow_to_openai_mcp

                    return workflow_to_openai_mcp(
                        self, input_schema, description, **kwargs
                    )
                case _ as unreachable:
                    assert_never(unreachable)
        elif isinstance(self, Standalone):
            match provider:
                case MCPProvider.CLAUDE:
                    from hatchet_sdk.runnables.mcp.claude import task_to_claude_mcp

                    return task_to_claude_mcp(self, input_schema, description, **kwargs)
                case MCPProvider.OPENAI:
                    from hatchet_sdk.runnables.mcp.openai import task_to_openai_mcp

                    return task_to_openai_mcp(self, input_schema, description, **kwargs)
                case _ as unreachable:
                    assert_never(unreachable)
        else:
            raise NotImplementedError()


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

    @overload
    def run(
        self,
        input: TWorkflowInput = ...,
        wait_for_result: Literal[True] = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> dict[str, Any]: ...

    @overload
    def run(
        self,
        input: TWorkflowInput = ...,
        *,
        wait_for_result: Literal[False],
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> WorkflowRunRef: ...

    def run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        wait_for_result: bool = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> WorkflowRunRef | dict[str, Any]:
        """
        Run the workflow synchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param wait_for_result: If True, block until completion and return the result. If False, return a WorkflowRunRef immediately.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the workflow run.
        :param sticky: Whether to use sticky scheduling for the workflow run.
        :param desired_worker_id: The ID of the desired worker to run the workflow on.
        :param desired_worker_labels: A list of desired worker labels for worker affinity.

        :returns: The result of the workflow execution as a dictionary, or a WorkflowRunRef if wait_for_result is False.
        """

        ref = self._client._client.admin.run_workflow(
            workflow_name=self._config.name,
            input=self._serialize_input(input, target="string"),
            options=self._create_trigger_run_options_with_combined_additional_meta(
                child_key=child_key,
                additional_metadata=additional_metadata,
                priority=priority,
                sticky=sticky,
                desired_worker_id=desired_worker_id,
                desired_worker_labels=desired_worker_labels,
            ),
        )

        if not wait_for_result:
            return ref

        return ref.result()

    @overload
    async def aio_run(
        self,
        input: TWorkflowInput = ...,
        wait_for_result: Literal[True] = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> dict[str, Any]: ...

    @overload
    async def aio_run(
        self,
        input: TWorkflowInput = ...,
        *,
        wait_for_result: Literal[False],
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> WorkflowRunRef: ...

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        wait_for_result: bool = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> WorkflowRunRef | dict[str, Any]:
        """
        Run the workflow asynchronously and wait for it to complete.

        This method triggers a workflow run, awaits until completion, and returns the final result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param wait_for_result: If True, await completion and return the result. If False, return a WorkflowRunRef immediately.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the workflow run.
        :param sticky: Whether to use sticky scheduling for the workflow run.
        :param desired_worker_id: The ID of the desired worker to run the workflow on.
        :param desired_worker_labels: A list of desired worker labels for worker affinity.

        :returns: The result of the workflow execution as a dictionary, or a WorkflowRunRef if wait_for_result is False.

        :raises RuntimeError: If the workflow is triggered within a durable context that supports durable eviction but fails to spawn a durable child workflow.
        """

        opts = self._create_trigger_run_options_with_combined_additional_meta(
            child_key=child_key,
            additional_metadata=additional_metadata,
            priority=priority,
            sticky=sticky,
            desired_worker_id=desired_worker_id,
            desired_worker_labels=desired_worker_labels,
        )

        durable_ctx = ctx_durable_context.get()
        if durable_ctx is not None and durable_ctx._supports_durable_eviction:
            config = WorkflowRunTriggerConfig(
                workflow_name=self._config.name,
                input=self._serialize_input(input, target="string"),
                options=opts,
            )
            refs = await durable_ctx._spawn_children_no_wait([config])
            if not refs:
                raise RuntimeError(
                    "Failed to spawn durable child workflow: no run references returned"
                )

            return await durable_ctx._aio_result_for_spawned_child(
                node_id=refs[0].node_id,
                branch_id=refs[0].branch_id,
                workflow_name=refs[0].workflow_name,
            )

        ref = await self._client._client.admin.aio_run_workflow(
            workflow_name=self._config.name,
            input=self._serialize_input(input, target="string"),
            options=opts,
        )

        if not wait_for_result:
            return ref

        return await ref.aio_result()

    def _get_result(
        self, ref: WorkflowRunRef, return_exceptions: bool
    ) -> dict[str, Any] | BaseException:
        try:
            return ref.result()
        except Exception as e:
            if return_exceptions:
                return e
            raise e

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[True],
        wait_for_result: Literal[True] = True,
    ) -> list[dict[str, Any] | BaseException]: ...

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[False] = False,
        wait_for_result: Literal[True] = True,
    ) -> list[dict[str, Any]]: ...

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        *,
        wait_for_result: Literal[False],
    ) -> list[WorkflowRunRef]: ...

    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: bool = False,
        wait_for_result: bool = True,
    ) -> (
        list[dict[str, Any]]
        | list[dict[str, Any] | BaseException]
        | list[WorkflowRunRef]
    ):
        """
        Run a workflow in bulk.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :param return_exceptions: If `True`, exceptions will be returned as part of the results instead of raising them.
        :param wait_for_result: If True, block until all runs complete and return results. If False, return a list of WorkflowRunRef immediately.
        :returns: A list of results for each workflow run, or a list of WorkflowRunRef if wait_for_result is False.
        """
        refs = self._client._client.admin.run_workflows(
            workflows=workflows,
        )

        if not wait_for_result:
            return refs

        return [self._get_result(ref, return_exceptions) for ref in refs]

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[True],
        wait_for_result: Literal[True] = True,
    ) -> list[dict[str, Any] | BaseException]: ...

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[False] = False,
        wait_for_result: Literal[True] = True,
    ) -> list[dict[str, Any]]: ...

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        *,
        wait_for_result: Literal[False],
    ) -> list[WorkflowRunRef]: ...

    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: bool = False,
        wait_for_result: bool = True,
    ) -> (
        list[dict[str, Any]]
        | list[dict[str, Any] | BaseException]
        | list[WorkflowRunRef]
    ):
        """
        Run a workflow in bulk asynchronously.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :param return_exceptions: If `True`, exceptions will be returned as part of the results instead of raising them.
        :param wait_for_result: If True, await completion and return results. If False, return a list of WorkflowRunRef immediately.
        :returns: A list of results for each workflow run, or a list of WorkflowRunRef if wait_for_result is False.
        """

        ## fixme: this might need a no-wait flavor?
        durable_ctx = ctx_durable_context.get()
        if durable_ctx is not None and durable_ctx._supports_durable_eviction:
            spawned_refs = await durable_ctx._spawn_children_no_wait(workflows)
            return await asyncio.gather(
                *[
                    durable_ctx._aio_result_for_spawned_child(
                        node_id=ref.node_id,
                        branch_id=ref.branch_id,
                        workflow_name=ref.workflow_name,
                    )
                    for ref in spawned_refs
                ],
                return_exceptions=return_exceptions,
            )

        refs = await self._client._client.admin.aio_run_workflows(
            workflows=workflows,
        )

        if not wait_for_result:
            return refs

        if return_exceptions:
            return await gather_max_concurrency(
                *[ref.aio_result() for ref in refs],
                return_exceptions=True,
                max_concurrency=10,
            )

        return await gather_max_concurrency(
            *[ref.aio_result() for ref in refs],
            return_exceptions=False,
            max_concurrency=10,
        )

    def _parse_task_name(
        self,
        name: str | None,
        func: Callable[..., Any],
    ) -> str:
        non_null_name = name or func.__name__
        return non_null_name.lower()

    def task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        parents: list[Task[TWorkflowInput, Any]] | None = None,
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: (
            dict[str, DesiredWorkerLabel] | list[DesiredWorkerLabel] | None
        ) = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: int | list[ConcurrencyExpression] | None = None,
        wait_for: list[Condition | OrGroup] | None = None,
        skip_if: list[Condition | OrGroup] | None = None,
        cancel_if: list[Condition | OrGroup] | None = None,
    ) -> Callable[
        [Callable[Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]]],
        Task[TWorkflowInput, R],
    ]:
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

        :param concurrency: A list of concurrency expressions for the task. If an integer is provided, it is treated as a constant concurrency limit with a `GROUP_ROUND_ROBIN` strategy, which means that only `N` runs of the task may execute at any given time.

        :param wait_for: A list of conditions that must be met before the task can run.

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped.

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled.

        :returns: A decorator which creates a `Task` object.
        """

        computed_params = ComputedTaskParameters(
            schedule_timeout=schedule_timeout,
            execution_timeout=execution_timeout,
            retries=retries,
            backoff_factor=backoff_factor,
            backoff_max_seconds=backoff_max_seconds,
            task_defaults=self._config.task_defaults,
        )

        def inner(
            func: Callable[
                Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]
            ],
        ) -> Task[TWorkflowInput, R]:
            labels: list[DesiredWorkerLabel] = (
                desired_worker_labels
                if isinstance(desired_worker_labels, list)
                else [
                    DesiredWorkerLabel(key=k, **d.model_dump(exclude={"key"}))
                    for k, d in (desired_worker_labels or {}).items()
                ]
            )

            task = Task(
                _fn=func,
                is_durable=False,
                workflow=self,
                type=StepType.DEFAULT,
                name=self._parse_task_name(name, func),
                execution_timeout=computed_params.execution_timeout,
                schedule_timeout=computed_params.schedule_timeout,
                parents=parents,
                retries=computed_params.retries,
                rate_limits=[r.to_proto() for r in rate_limits or []],
                desired_worker_labels=labels,
                backoff_factor=computed_params.backoff_factor,
                backoff_max_seconds=computed_params.backoff_max_seconds,
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
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        parents: list[Task[TWorkflowInput, Any]] | None = None,
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: (
            dict[str, DesiredWorkerLabel] | list[DesiredWorkerLabel] | None
        ) = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: int | list[ConcurrencyExpression] | None = None,
        wait_for: list[Condition | OrGroup] | None = None,
        skip_if: list[Condition | OrGroup] | None = None,
        cancel_if: list[Condition | OrGroup] | None = None,
        eviction_policy: EvictionPolicy | None = DEFAULT_DURABLE_TASK_EVICTION_POLICY,
    ) -> Callable[
        [
            Callable[
                Concatenate[TWorkflowInput, DurableContext, P], R | CoroutineLike[R]
            ]
        ],
        Task[TWorkflowInput, R],
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

        :param concurrency: A list of concurrency expressions for the task. If an integer is provided, it is treated as a constant concurrency limit with a `GROUP_ROUND_ROBIN` strategy, which means that only `N` runs of the task may execute at any given time.

        :param wait_for: A list of conditions that must be met before the task can run.

        :param skip_if: A list of conditions that, if met, will cause the task to be skipped.

        :param cancel_if: A list of conditions that, if met, will cause the task to be canceled.

        :param eviction_policy: An optional eviction policy controlling when this durable task can be evicted from a worker slot while waiting.

        :returns: A decorator which creates a `Task` object.
        """

        computed_params = ComputedTaskParameters(
            schedule_timeout=schedule_timeout,
            execution_timeout=execution_timeout,
            retries=retries,
            backoff_factor=backoff_factor,
            backoff_max_seconds=backoff_max_seconds,
            task_defaults=self._config.task_defaults,
        )

        def inner(
            func: Callable[
                Concatenate[TWorkflowInput, DurableContext, P], R | CoroutineLike[R]
            ],
        ) -> Task[TWorkflowInput, R]:
            labels: list[DesiredWorkerLabel] = (
                desired_worker_labels
                if isinstance(desired_worker_labels, list)
                else [
                    DesiredWorkerLabel(key=k, **d.model_dump(exclude={"key"}))
                    for k, d in (desired_worker_labels or {}).items()
                ]
            )

            task = Task(
                _fn=func,
                is_durable=True,
                workflow=self,
                type=StepType.DEFAULT,
                name=self._parse_task_name(name, func),
                execution_timeout=computed_params.execution_timeout,
                schedule_timeout=computed_params.schedule_timeout,
                parents=parents,
                retries=computed_params.retries,
                rate_limits=[r.to_proto() for r in rate_limits or []],
                desired_worker_labels=labels,
                backoff_factor=computed_params.backoff_factor,
                backoff_max_seconds=computed_params.backoff_max_seconds,
                concurrency=concurrency,
                wait_for=wait_for,
                skip_if=skip_if,
                cancel_if=cancel_if,
                eviction_policy=eviction_policy,
            )

            self._durable_tasks.append(task)

            return task

        return inner

    def on_failure_task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: int | list[ConcurrencyExpression] | None = None,
    ) -> Callable[
        [Callable[Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]]],
        Task[TWorkflowInput, R],
    ]:
        """
        A decorator to transform a function into a Hatchet on-failure task that runs as the last step in a workflow that had at least one task fail.

        :param name: The name of the on-failure task. If not specified, defaults to the name of the function being wrapped by the `on_failure_task` decorator.

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param retries: The number of times to retry the on-failure task before failing.

        :param rate_limits: A list of rate limit configurations for the on-failure task.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the on-failure task. If an integer is provided, it is treated as a constant concurrency limit with a `GROUP_ROUND_ROBIN` strategy, which means that only `N` runs of the task may execute at any given time.

        :returns: A decorator which creates a `Task` object.
        """

        def inner(
            func: Callable[
                Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]
            ],
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
                rate_limits=[r.to_proto() for r in rate_limits or []],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
                desired_worker_labels=None,
                parents=None,
                wait_for=None,
                skip_if=None,
                cancel_if=None,
            )

            if self._on_failure_task:
                raise ValueError("Only one on-failure task is allowed")

            self._on_failure_task = task

            return task

        return inner

    def on_success_task(
        self,
        name: str | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        concurrency: int | list[ConcurrencyExpression] | None = None,
    ) -> Callable[
        [Callable[Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]]],
        Task[TWorkflowInput, R],
    ]:
        """
        A decorator to transform a function into a Hatchet on-success task that runs as the last step in a workflow that had all upstream tasks succeed.

        :param name: The name of the on-success task. If not specified, defaults to the name of the function being wrapped by the `on_success_task` decorator.

        :param schedule_timeout: The maximum time to wait for the task to be scheduled. The run will be canceled if the task does not begin within this time.

        :param execution_timeout: The maximum time to wait for the task to complete. The run will be canceled if the task does not complete within this time.

        :param retries: The number of times to retry the on-success task before failing

        :param rate_limits: A list of rate limit configurations for the on-success task.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param concurrency: A list of concurrency expressions for the on-success task. If an integer is provided, it is treated as a constant concurrency limit with a `GROUP_ROUND_ROBIN` strategy, which means that only `N` runs of the task may execute at any given time.

        :returns: A decorator which creates a Task object.
        """

        def inner(
            func: Callable[
                Concatenate[TWorkflowInput, Context, P], R | CoroutineLike[R]
            ],
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
                rate_limits=[r.to_proto() for r in rate_limits or []],
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=concurrency,
                parents=None,
                desired_worker_labels=None,
                wait_for=None,
                skip_if=None,
                cancel_if=None,
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


class TaskRunRef(Generic[TWorkflowInput, R]):
    def __init__(
        self,
        standalone: "Standalone[TWorkflowInput, R]",
        workflow_run_ref: WorkflowRunRef,
    ):
        self._s = standalone
        self._wrr = workflow_run_ref

        self.workflow_run_id = workflow_run_ref.workflow_run_id

    def __str__(self) -> str:
        return self.workflow_run_id

    async def aio_result(self) -> R:
        result = await self._wrr._workflow_run_listener.aio_result(
            self._wrr.workflow_run_id
        )
        return self._s._extract_result(result)

    def result(self) -> R:
        result = self._wrr.result()

        return self._s._extract_result(result)

    def stream(self) -> RunEventListener:
        return self._wrr._stream()


class Standalone(BaseWorkflow[TWorkflowInput], Generic[TWorkflowInput, R]):
    def __init__(
        self, workflow: Workflow[TWorkflowInput], task: Task[TWorkflowInput, R]
    ) -> None:
        super().__init__(config=workflow._config, client=workflow._client)

        ## NOTE: This is a hack to assign the task back to the base workflow,
        ## since the decorator to mutate the tasks is not being called.
        self._default_tasks = [task]

        self._workflow = workflow
        self._task = task

        return_type = get_type_hints(self._task._fn).get("return")

        self._output_validator: TypeAdapter[TaskPayloadForInternalUse] = TypeAdapter(
            normalize_validator(return_type)
        )

    @overload
    def _extract_result(self, result: dict[str, Any]) -> R: ...

    @overload
    def _extract_result(self, result: BaseException) -> BaseException: ...

    def _extract_result(
        self, result: dict[str, Any] | BaseException
    ) -> R | BaseException:
        if isinstance(result, BaseException):
            return result

        # if a task is cancelled, we can get `None` back here
        ## this is a bit of an edge case since both `None` and an empty dict
        ## would cause Pydantic validation errors, but if you were expecting a `dict`
        ## return, then the empty dict would not error and would work correctly

        # Durable child callbacks can return the task payload directly, while
        # non-durable child runs typically return {task_name: payload}.
        output = result.get(self._task.name) or result or {}

        return cast(
            R,
            self._output_validator.validate_python(
                output, context=HATCHET_PYDANTIC_SENTINEL
            ),
        )

    @overload
    def run(
        self,
        input: TWorkflowInput = ...,
        wait_for_result: Literal[True] = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> R: ...

    @overload
    def run(
        self,
        input: TWorkflowInput = ...,
        *,
        wait_for_result: Literal[False],
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> TaskRunRef[TWorkflowInput, R]: ...

    def run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        wait_for_result: bool = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> TaskRunRef[TWorkflowInput, R] | R:
        """
        Run the workflow synchronously and wait for it to complete.

        This method triggers a workflow run, blocks until completion, and returns the extracted result.

        :param input: The input data for the workflow.
        :param wait_for_result: If True, block until completion and return the result. If False, return a TaskRunRef immediately.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the workflow run.
        :param sticky: Whether to use sticky scheduling for the workflow run.
        :param desired_worker_id: The ID of the desired worker to run the workflow on.
        :param desired_worker_labels: A list of desired worker labels for worker affinity.

        :returns: The extracted result of the workflow execution, or a TaskRunRef if wait_for_result is False.
        """
        if not wait_for_result:
            ref = self._workflow.run(
                input,
                wait_for_result=False,
                child_key=child_key,
                additional_metadata=additional_metadata,
                priority=priority,
                sticky=sticky,
                desired_worker_id=desired_worker_id,
                desired_worker_labels=desired_worker_labels,
            )
            return TaskRunRef[TWorkflowInput, R](self, ref)

        return self._extract_result(
            self._workflow.run(
                input,
                wait_for_result=True,
                child_key=child_key,
                additional_metadata=additional_metadata,
                priority=priority,
                sticky=sticky,
                desired_worker_id=desired_worker_id,
                desired_worker_labels=desired_worker_labels,
            )
        )

    @overload
    async def aio_run(
        self,
        input: TWorkflowInput = ...,
        wait_for_result: Literal[True] = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> R: ...

    @overload
    async def aio_run(
        self,
        input: TWorkflowInput = ...,
        *,
        wait_for_result: Literal[False],
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> TaskRunRef[TWorkflowInput, R]: ...

    async def aio_run(
        self,
        input: TWorkflowInput = cast(TWorkflowInput, EmptyModel()),
        wait_for_result: bool = True,
        child_key: str | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        priority: int | None = None,
        sticky: bool = False,
        desired_worker_id: str | None = None,
        desired_worker_labels: list[DesiredWorkerLabel] | None = None,
    ) -> TaskRunRef[TWorkflowInput, R] | R:
        """
        Run the workflow asynchronously and wait for it to complete.

        This method triggers a workflow run, awaits until completion, and returns the extracted result.

        :param input: The input data for the workflow, must match the workflow's input type.
        :param wait_for_result: If True, await completion and return the result. If False, return a TaskRunRef immediately.
        :param child_key: An optional key for deduplicating child workflow runs.
        :param additional_metadata: Additional metadata to attach to the workflow run.
        :param priority: The priority of the workflow run.
        :param sticky: Whether to use sticky scheduling for the workflow run.
        :param desired_worker_id: The ID of the desired worker to run the workflow on.
        :param desired_worker_labels: A list of desired worker labels for worker affinity.

        :returns: The extracted result of the workflow execution, or a TaskRunRef if wait_for_result is False.
        """

        if not wait_for_result:
            ref = await self._workflow.aio_run(
                input,
                wait_for_result=False,
                child_key=child_key,
                additional_metadata=additional_metadata,
                priority=priority,
                sticky=sticky,
                desired_worker_id=desired_worker_id,
                desired_worker_labels=desired_worker_labels,
            )
            return TaskRunRef[TWorkflowInput, R](self, ref)

        res = await self._workflow.aio_run(
            input,
            wait_for_result=True,
            child_key=child_key,
            additional_metadata=additional_metadata,
            priority=priority,
            sticky=sticky,
            desired_worker_id=desired_worker_id,
            desired_worker_labels=desired_worker_labels,
        )
        return await asyncio.to_thread(self._extract_result, res)

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[True],
        wait_for_result: Literal[True] = True,
    ) -> list[R | BaseException]: ...

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[False] = False,
        wait_for_result: Literal[True] = True,
    ) -> list[R]: ...

    @overload
    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        *,
        wait_for_result: Literal[False],
    ) -> list[TaskRunRef[TWorkflowInput, R]]: ...

    def run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: bool = False,
        wait_for_result: bool = True,
    ) -> list[R] | list[R | BaseException] | list[TaskRunRef[TWorkflowInput, R]]:
        """
        Run a workflow in bulk.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :param return_exceptions: If `True`, exceptions will be returned as part of the results instead of raising them.
        :param wait_for_result: If True, block until all runs complete and return results. If False, return a list of TaskRunRef immediately.
        :returns: A list of results for each workflow run, or a list of TaskRunRef if wait_for_result is False.
        """
        if not wait_for_result:
            refs = self._workflow.run_many(workflows, wait_for_result=False)
            return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]

        return [
            self._extract_result(result)
            for result in self._workflow.run_many(
                workflows,
                ## hack: typing needs literal
                True if return_exceptions else False,  # noqa: SIM210
                wait_for_result=True,
            )
        ]

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[True],
        wait_for_result: Literal[True] = True,
    ) -> list[R | BaseException]: ...

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: Literal[False] = False,
        wait_for_result: Literal[True] = True,
    ) -> list[R]: ...

    @overload
    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        *,
        wait_for_result: Literal[False],
    ) -> list[TaskRunRef[TWorkflowInput, R]]: ...

    async def aio_run_many(
        self,
        workflows: list[WorkflowRunTriggerConfig],
        return_exceptions: bool = False,
        wait_for_result: bool = True,
    ) -> list[R] | list[R | BaseException] | list[TaskRunRef[TWorkflowInput, R]]:
        """
        Run a workflow in bulk asynchronously.

        :param workflows: A list of `WorkflowRunTriggerConfig` objects, each representing a workflow run to be triggered.
        :param return_exceptions: If `True`, exceptions will be returned as part of the results instead of raising them.
        :param wait_for_result: If True, await completion and return results. If False, return a list of TaskRunRef immediately.
        :returns: A list of results for each workflow run, or a list of TaskRunRef if wait_for_result is False.
        """
        if not wait_for_result:
            refs = await self._workflow.aio_run_many(workflows, wait_for_result=False)
            return [TaskRunRef[TWorkflowInput, R](self, ref) for ref in refs]

        return [
            self._extract_result(result)
            for result in await self._workflow.aio_run_many(
                workflows,
                ## hack: typing needs literal
                True if return_exceptions else False,  # noqa: SIM210
                wait_for_result=True,
            )
        ]

    def mock_run(
        self,
        input: TWorkflowInput | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        parent_outputs: dict[str, JSONSerializableMapping] | None = None,
        retry_count: int = 0,
        lifespan: Any = None,
        dependencies: dict[str, Any] | None = None,
    ) -> R:
        """
        Mimic the execution of a task. This method is intended to be used to unit test
        tasks without needing to interact with the Hatchet engine. Use `mock_run` for sync
        tasks and `aio_mock_run` for async tasks.

        :param input: The input to the task.
        :param additional_metadata: Additional metadata to attach to the task.
        :param parent_outputs: Outputs from parent tasks, if any. This is useful for mimicking DAG functionality. For instance, if you have a task `step_2` that has a `parent` which is `step_1`, you can pass `parent_outputs={"step_1": {"result": "Hello, world!"}}` to `step_2.mock_run()` to be able to access `ctx.task_output(step_1)` in `step_2`.
        :param retry_count: The number of times the task has been retried.
        :param lifespan: The lifespan to be used in the task, which is useful if one was set on the worker. This will allow you to access `ctx.lifespan` inside of your task.
        :param dependencies: Dependencies to be injected into the task. This is useful for tasks that have dependencies defined using `Depends`. **IMPORTANT**: You must pass the dependencies _directly_, **not** the `Depends` objects themselves. For example, if you have a task that has a dependency `config: Annotated[str, Depends(get_config)]`, you should pass `dependencies={"config": "config_value"}` to `aio_mock_run`.

        :return: The output of the task.
        """

        return self._task.mock_run(
            input=input,
            additional_metadata=additional_metadata,
            parent_outputs=parent_outputs,
            retry_count=retry_count,
            lifespan=lifespan,
            dependencies=dependencies,
        )

    async def aio_mock_run(
        self,
        input: TWorkflowInput | None = None,
        additional_metadata: JSONSerializableMapping | None = None,
        parent_outputs: dict[str, JSONSerializableMapping] | None = None,
        retry_count: int = 0,
        lifespan: Any = None,
        dependencies: dict[str, Any] | None = None,
    ) -> R:
        """
        Mimic the execution of a task. This method is intended to be used to unit test
        tasks without needing to interact with the Hatchet engine. Use `mock_run` for sync
        tasks and `aio_mock_run` for async tasks.

        :param input: The input to the task.
        :param additional_metadata: Additional metadata to attach to the task.
        :param parent_outputs: Outputs from parent tasks, if any. This is useful for mimicking DAG functionality. For instance, if you have a task `step_2` that has a `parent` which is `step_1`, you can pass `parent_outputs={"step_1": {"result": "Hello, world!"}}` to `step_2.mock_run()` to be able to access `ctx.task_output(step_1)` in `step_2`.
        :param retry_count: The number of times the task has been retried.
        :param lifespan: The lifespan to be used in the task, which is useful if one was set on the worker. This will allow you to access `ctx.lifespan` inside of your task.
        :param dependencies: Dependencies to be injected into the task. This is useful for tasks that have dependencies defined using `Depends`. **IMPORTANT**: You must pass the dependencies _directly_, **not** the `Depends` objects themselves. For example, if you have a task that has a dependency `config: Annotated[str, Depends(get_config)]`, you should pass `dependencies={"config": "config_value"}` to `aio_mock_run`.

        :return: The output of the task.
        """

        return await self._task.aio_mock_run(
            input=input,
            additional_metadata=additional_metadata,
            parent_outputs=parent_outputs,
            retry_count=retry_count,
            lifespan=lifespan,
            dependencies=dependencies,
        )

    @property
    def is_async_function(self) -> bool:
        """
        Check if the task is an async function.

        :returns: True if the task is an async function, False otherwise.
        """
        return self._task._is_async_function

    def get_run_ref(self, run_id: str) -> TaskRunRef[TWorkflowInput, R]:
        """
        Get a reference to a task run by its run ID.

        :param run_id: The ID of the run to get the reference for.
        :returns: A `TaskRunRef` object representing the reference to the task run.
        """
        wrr = self._workflow._client._client.runs.get_run_ref(run_id)
        return TaskRunRef[TWorkflowInput, R](self, wrr)

    async def aio_get_result(self, run_id: str) -> R:
        """
        Get the result of a task run by its run ID.

        :param run_id: The ID of the run to get the result for.
        :returns: The result of the task run.
        """
        run_ref = self.get_run_ref(run_id)

        return await run_ref.aio_result()

    def get_result(self, run_id: str) -> R:
        """
        Get the result of a task run by its run ID.

        :param run_id: The ID of the run to get the result for.
        :returns: The result of the task run.
        """
        run_ref = self.get_run_ref(run_id)

        return run_ref.result()

    @property
    def output_validator(self) -> TypeAdapter[R]:
        return cast(TypeAdapter[R], self._output_validator)

    @property
    def output_validator_type(self) -> type[R]:
        return cast(type[R], self._output_validator._type)
