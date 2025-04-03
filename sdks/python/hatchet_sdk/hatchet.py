import asyncio
import logging
from typing import Any, Callable, Type, cast, overload

from hatchet_sdk import Context, DurableContext
from hatchet_sdk.client import Client
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.features.cron import CronClient
from hatchet_sdk.features.logs import LogsClient
from hatchet_sdk.features.metrics import MetricsClient
from hatchet_sdk.features.rate_limits import RateLimitsClient
from hatchet_sdk.features.runs import RunsClient
from hatchet_sdk.features.scheduled import ScheduledClient
from hatchet_sdk.features.workers import WorkersClient
from hatchet_sdk.features.workflows import WorkflowsClient
from hatchet_sdk.labels import DesiredWorkerLabel
from hatchet_sdk.logger import logger
from hatchet_sdk.rate_limit import RateLimit
from hatchet_sdk.runnables.standalone import Standalone
from hatchet_sdk.runnables.types import (
    DEFAULT_EXECUTION_TIMEOUT,
    DEFAULT_SCHEDULE_TIMEOUT,
    ConcurrencyExpression,
    EmptyModel,
    R,
    StickyStrategy,
    TaskDefaults,
    TWorkflowInput,
    WorkflowConfig,
)
from hatchet_sdk.runnables.workflow import BaseWorkflow, Workflow
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.worker.worker import Worker


class Hatchet:
    """
    Main client for interacting with the Hatchet SDK.

    This class provides access to various client interfaces and utility methods
    for working with Hatchet workers, workflows, and steps.

    Attributes:
        cron (CronClient): Interface for cron trigger operations.

        admin (AdminClient): Interface for administrative operations.
        dispatcher (DispatcherClient): Interface for dispatching operations.
        event (EventClient): Interface for event-related operations.
        rest (RestApi): Interface for REST API operations.
    """

    def __init__(
        self,
        debug: bool = False,
        client: Client | None = None,
        config: ClientConfig | None = None,
    ):
        """
        Initialize a new Hatchet instance.

        :param debug: Enable debug logging. Default: `False`
        :type debug: bool

        :param client: A pre-configured `Client` instance. Default: `None`.
        :type client: Client | None

        :param config: Configuration for creating a new Client. Defaults to ClientConfig()
        :type config: ClientConfig
        """

        if debug:
            logger.setLevel(logging.DEBUG)

        self._client = (
            client if client else Client(config=config or ClientConfig(), debug=debug)
        )

    @property
    def cron(self) -> CronClient:
        return self._client.cron

    @property
    def logs(self) -> LogsClient:
        return self._client.logs

    @property
    def metrics(self) -> MetricsClient:
        return self._client.metrics

    @property
    def rate_limits(self) -> RateLimitsClient:
        return self._client.rate_limits

    @property
    def runs(self) -> RunsClient:
        return self._client.runs

    @property
    def scheduled(self) -> ScheduledClient:
        return self._client.scheduled

    @property
    def workers(self) -> WorkersClient:
        return self._client.workers

    @property
    def workflows(self) -> WorkflowsClient:
        return self._client.workflows

    @property
    def dispatcher(self) -> DispatcherClient:
        return self._client.dispatcher

    @property
    def event(self) -> EventClient:
        return self._client.event

    @property
    def listener(self) -> RunEventListenerClient:
        return self._client.listener

    @property
    def config(self) -> ClientConfig:
        return self._client.config

    @property
    def tenant_id(self) -> str:
        return self._client.config.tenant_id

    def worker(
        self,
        name: str,
        slots: int = 100,
        labels: dict[str, str | int] = {},
        workflows: list[BaseWorkflow[Any]] = [],
    ) -> Worker:
        """
        Create a Hatchet worker on which to run workflows.

        :param name: The name of the worker.
        :type name: str

        :param slots: The number of workflow slots on the worker. In other words, the number of concurrent tasks the worker can run at any point in time. Default: 100
        :type slots: int

        :param labels: A dictionary of labels to assign to the worker. For more details, view examples on affinity and worker labels. Defaults to an empty dictionary (no labels)
        :type labels: dict[str, str | int]

        :param workflows: A list of workflows to register on the worker, as a shorthand for calling `register_workflow` on each or `register_workflows` on all of them. Defaults to an empty list
        :type workflows: list[Workflow]


        :returns: The created `Worker` object, which exposes an instance method `start` which can be called to start the worker.
        :rtype: Worker
        """

        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = None

        return Worker(
            name=name,
            slots=slots,
            labels=labels,
            config=self._client.config,
            debug=self._client.debug,
            owned_loop=loop is None,
            workflows=workflows,
        )

    @overload
    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
    ) -> Workflow[EmptyModel]: ...

    @overload
    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput],
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
    ) -> Workflow[TWorkflowInput]: ...

    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput] | None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
        """
        Define a Hatchet workflow, which can then declare `task`s and be `run`, `schedule`d, and so on.

        :param name: The name of the workflow.
        :type name: str

        :param description: A description for the workflow. Default: None
        :type description: str | None

        :param version: A version for the workflow. Default: None
        :type version: str | None

        :param input_validator: A Pydantic model to use as a validator for the `input` to the tasks in the workflow. If no validator is provided, defaults to an `EmptyModel` under the hood. The `EmptyModel` is a Pydantic model with no fields specified, and with the `extra` config option set to `"allow"`.
        :type input_validator: Type[BaseModel]

        :param on_events: A list of event triggers for the workflow - events which cause the workflow to be run. Defaults to an empty list, meaning the workflow will not be run on any event pushes.
        :type on_events: list[str]

        :param on_crons: A list of cron triggers for the workflow. Defaults to an empty list, meaning the workflow will not be run on any cron schedules.
        :type on_crons: list[str]

        :param sticky: A sticky strategy for the workflow. Default: `None`
        :type sticky: StickyStategy

        :param default_priority: The priority of the workflow. Higher values will cause this workflow to have priority in scheduling over other, lower priority ones. Default: `1`
        :type default_priority: int

        :param concurrency: A concurrency object controlling the concurrency settings for this workflow.
        :type concurrency: ConcurrencyExpression | None

        :param task_defaults: A `TaskDefaults` object controlling the default task settings for this workflow.
        :type task_defaults: TaskDefaults

        :returns: The created `Workflow` object, which can be used to declare tasks, run the workflow, and so on.
        :rtype: Workflow
        """

        return Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                version=version,
                description=description,
                on_events=on_events,
                on_crons=on_crons,
                sticky=sticky,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(Type[TWorkflowInput], EmptyModel),
                task_defaults=task_defaults,
            ),
            self,
        )

    @overload
    def task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[[Callable[[EmptyModel, Context], R]], Standalone[EmptyModel, R]]: ...

    @overload
    def task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput],
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[
        [Callable[[TWorkflowInput, Context], R]], Standalone[TWorkflowInput, R]
    ]: ...

    def task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput] | None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> (
        Callable[[Callable[[EmptyModel, Context], R]], Standalone[EmptyModel, R]]
        | Callable[
            [Callable[[TWorkflowInput, Context], R]], Standalone[TWorkflowInput, R]
        ]
    ):
        """
        A decorator to transform a function into a standalone Hatchet task that runs as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.
        :type name: str

        :param description: An optional description for the task. Default: None
        :type description: str | None

        :param input_validator: A Pydantic model to use as a validator for the input to the task. If no validator is provided, defaults to an `EmptyModel`.
        :type input_validator: Type[BaseModel]

        :param on_events: A list of event triggers for the task - events which cause the task to be run. Defaults to an empty list.
        :type on_events: list[str]

        :param on_crons: A list of cron triggers for the task. Defaults to an empty list.
        :type on_crons: list[str]

        :param version: A version for the task. Default: None
        :type version: str | None

        :param sticky: A sticky strategy for the task. Default: None
        :type sticky: StickyStrategy | None

        :param default_priority: The priority of the task. Higher values will cause this task to have priority in scheduling. Default: 1
        :type default_priority: int

        :param concurrency: A concurrency object controlling the concurrency settings for this task.
        :type concurrency: ConcurrencyExpression | None

        :param schedule_timeout: The maximum time allowed for scheduling the task. Default: DEFAULT_SCHEDULE_TIMEOUT
        :type schedule_timeout: Duration

        :param execution_timeout: The maximum time allowed for executing the task. Default: DEFAULT_EXECUTION_TIMEOUT
        :type execution_timeout: Duration

        :param retries: The number of times to retry the task before failing. Default: 0
        :type retries: int

        :param rate_limits: A list of rate limit configurations for the task. Defaults to an empty list.
        :type rate_limits: list[RateLimit]

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned.
        :type desired_worker_labels: dict[str, DesiredWorkerLabel]

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries. Default: None
        :type backoff_factor: float | None

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue. Default: None
        :type backoff_max_seconds: int | None

        :returns: A decorator which creates a `Standalone` task object.
        :rtype: Callable[[Callable[[TWorkflowInput, Context], R]], Standalone[TWorkflowInput, R]]
        """

        workflow = Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                version=version,
                description=description,
                on_events=on_events,
                on_crons=on_crons,
                sticky=sticky,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(Type[TWorkflowInput], EmptyModel),
            ),
            self,
        )

        task_wrapper = workflow.task(
            name=name,
            schedule_timeout=schedule_timeout,
            execution_timeout=execution_timeout,
            parents=[],
            retries=retries,
            rate_limits=rate_limits,
            desired_worker_labels=desired_worker_labels,
            backoff_factor=backoff_factor,
            backoff_max_seconds=backoff_max_seconds,
            concurrency=[concurrency] if concurrency else [],
        )

        def inner(
            func: Callable[[TWorkflowInput, Context], R]
        ) -> Standalone[TWorkflowInput, R]:
            created_task = task_wrapper(func)

            return Standalone[TWorkflowInput, R](
                workflow=workflow,
                task=created_task,
            )

        return inner

    @overload
    def durable_task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[
        [Callable[[EmptyModel, DurableContext], R]], Standalone[EmptyModel, R]
    ]: ...

    @overload
    def durable_task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput],
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> Callable[
        [Callable[[TWorkflowInput, DurableContext], R]], Standalone[TWorkflowInput, R]
    ]: ...

    def durable_task(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: Type[TWorkflowInput] | None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
        schedule_timeout: Duration = DEFAULT_SCHEDULE_TIMEOUT,
        execution_timeout: Duration = DEFAULT_EXECUTION_TIMEOUT,
        retries: int = 0,
        rate_limits: list[RateLimit] = [],
        desired_worker_labels: dict[str, DesiredWorkerLabel] = {},
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
    ) -> (
        Callable[[Callable[[EmptyModel, DurableContext], R]], Standalone[EmptyModel, R]]
        | Callable[
            [Callable[[TWorkflowInput, DurableContext], R]],
            Standalone[TWorkflowInput, R],
        ]
    ):
        """
        A decorator to transform a function into a standalone Hatchet _durable_ task that runs as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.
        :type name: str

        :param description: An optional description for the task. Default: None
        :type description: str | None

        :param input_validator: A Pydantic model to use as a validator for the input to the task. If no validator is provided, defaults to an `EmptyModel`.
        :type input_validator: Type[BaseModel]

        :param on_events: A list of event triggers for the task - events which cause the task to be run. Defaults to an empty list.
        :type on_events: list[str]

        :param on_crons: A list of cron triggers for the task. Defaults to an empty list.
        :type on_crons: list[str]

        :param version: A version for the task. Default: None
        :type version: str | None

        :param sticky: A sticky strategy for the task. Default: None
        :type sticky: StickyStrategy | None

        :param default_priority: The priority of the task. Higher values will cause this task to have priority in scheduling. Default: 1
        :type default_priority: int

        :param concurrency: A concurrency object controlling the concurrency settings for this task.
        :type concurrency: ConcurrencyExpression | None

        :param schedule_timeout: The maximum time allowed for scheduling the task. Default: DEFAULT_SCHEDULE_TIMEOUT
        :type schedule_timeout: Duration

        :param execution_timeout: The maximum time allowed for executing the task. Default: DEFAULT_EXECUTION_TIMEOUT
        :type execution_timeout: Duration

        :param retries: The number of times to retry the task before failing. Default: 0
        :type retries: int

        :param rate_limits: A list of rate limit configurations for the task. Defaults to an empty list.
        :type rate_limits: list[RateLimit]

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned.
        :type desired_worker_labels: dict[str, DesiredWorkerLabel]

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries. Default: None
        :type backoff_factor: float | None

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue. Default: None
        :type backoff_max_seconds: int | None

        :returns: A decorator which creates a `Standalone` task object.
        :rtype: Callable[[Callable[[TWorkflowInput, Context], R]], Standalone[TWorkflowInput, R]]
        """

        workflow = Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                version=version,
                description=description,
                on_events=on_events,
                on_crons=on_crons,
                sticky=sticky,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(Type[TWorkflowInput], EmptyModel),
            ),
            self,
        )

        task_wrapper = workflow.durable_task(
            name=name,
            schedule_timeout=schedule_timeout,
            execution_timeout=execution_timeout,
            parents=[],
            retries=retries,
            rate_limits=rate_limits,
            desired_worker_labels=desired_worker_labels,
            backoff_factor=backoff_factor,
            backoff_max_seconds=backoff_max_seconds,
            concurrency=[concurrency] if concurrency else [],
        )

        def inner(
            func: Callable[[TWorkflowInput, DurableContext], R]
        ) -> Standalone[TWorkflowInput, R]:
            created_task = task_wrapper(func)

            return Standalone[TWorkflowInput, R](
                workflow=workflow,
                task=created_task,
            )

        return inner
