import asyncio
import logging
from collections.abc import Callable
from datetime import timedelta
from functools import cached_property
from typing import Any, cast, overload

from hatchet_sdk import Context, DurableContext
from hatchet_sdk.client import Client
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.listeners.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.rest.models.tenant_version import TenantVersion
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.features.cron import CronClient
from hatchet_sdk.features.filters import FiltersClient
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
from hatchet_sdk.runnables.types import (
    ConcurrencyExpression,
    DefaultFilter,
    EmptyModel,
    R,
    StickyStrategy,
    TaskDefaults,
    TWorkflowInput,
    WorkflowConfig,
)
from hatchet_sdk.runnables.workflow import BaseWorkflow, Standalone, Workflow
from hatchet_sdk.utils.timedelta_to_expression import Duration
from hatchet_sdk.utils.typing import CoroutineLike
from hatchet_sdk.worker.worker import LifespanFn, Worker


class Hatchet:
    """
    Main client for interacting with the Hatchet SDK.

    This class provides access to various client interfaces and utility methods
    for working with Hatchet workers, workflows, tasks, and our various feature clients.
    """

    def __init__(
        self,
        debug: bool = False,
        client: Client | None = None,
        config: ClientConfig | None = None,
    ):
        if debug:
            logger.setLevel(logging.DEBUG)

        self._client = (
            client if client else Client(config=config or ClientConfig(), debug=debug)
        )

        if self.tenant_engine_version != TenantVersion.V1:
            logger.warning(
                "🚨⚠️‼️ YOU ARE USING A V0 ENGINE WITH A V1 SDK, WHICH IS NOT SUPPORTED. PLEASE UPGRADE YOUR ENGINE TO V1.🚨⚠️‼️"
            )

    @property
    def cron(self) -> CronClient:
        """
        The cron client is a client for managing cron workflows within Hatchet.
        """
        return self._client.cron

    @property
    def filters(self) -> FiltersClient:
        """
        The filters client is a client for interacting with Hatchet's filters API.
        """
        return self._client.filters

    @property
    def logs(self) -> LogsClient:
        """
        The logs client is a client for interacting with Hatchet's logs API.
        """
        return self._client.logs

    @property
    def metrics(self) -> MetricsClient:
        """
        The metrics client is a client for reading metrics out of Hatchet's metrics API.
        """
        return self._client.metrics

    @property
    def rate_limits(self) -> RateLimitsClient:
        """
        The rate limits client is a wrapper for Hatchet's gRPC API that makes it easier to work with rate limits in Hatchet.
        """
        return self._client.rate_limits

    @property
    def runs(self) -> RunsClient:
        """
        The runs client is a client for interacting with task and workflow runs within Hatchet.
        """
        return self._client.runs

    @property
    def scheduled(self) -> ScheduledClient:
        """
        The scheduled client is a client for managing scheduled workflows within Hatchet.
        """
        return self._client.scheduled

    @property
    def workers(self) -> WorkersClient:
        """
        The workers client is a client for managing workers programmatically within Hatchet.
        """
        return self._client.workers

    @property
    def workflows(self) -> WorkflowsClient:
        """
        The workflows client is a client for managing workflows programmatically within Hatchet.

        Note that workflows are the declaration, _not_ the individual runs. If you're looking for runs, use the `RunsClient` instead.
        """
        return self._client.workflows

    @property
    def dispatcher(self) -> DispatcherClient:
        return self._client.dispatcher

    @property
    def event(self) -> EventClient:
        """
        The event client, which you can use to push events to Hatchet.
        """
        return self._client.event

    @property
    def listener(self) -> RunEventListenerClient:
        return self._client.listener

    @property
    def config(self) -> ClientConfig:
        return self._client.config

    @property
    def tenant_id(self) -> str:
        """
        The tenant id you're operating in.
        """
        return self._client.config.tenant_id

    @property
    def namespace(self) -> str:
        """
        The current namespace you're interacting with.
        """
        return self._client.config.namespace

    @cached_property
    def tenant_engine_version(self) -> TenantVersion:
        """
        Get the version of the Hatchet engine running in your tenant.
        """
        try:
            return self._client.tenant.get().version
        except Exception:
            ## Nothing we can do here - if this fails, it's probably
            ## because they don't have this endpoint yet, so we need to just
            ## assume V1 to swallow the warning.
            return TenantVersion.V1

    def worker(
        self,
        name: str,
        slots: int = 100,
        durable_slots: int = 1_000,
        labels: dict[str, str | int] | None = None,
        workflows: list[BaseWorkflow[Any]] | None = None,
        lifespan: LifespanFn | None = None,
    ) -> Worker:
        """
        Create a Hatchet worker on which to run workflows.

        :param name: The name of the worker.

        :param slots: The number of workflow slots on the worker. In other words, the number of concurrent tasks the worker can run at any point in time

        :param durable_slots: The number of durable workflow slots on the worker. In other words, the number of concurrent tasks the worker can run at any point in time that are durable.

        :param labels: A dictionary of labels to assign to the worker. For more details, view examples on affinity and worker labels.

        :param workflows: A list of workflows to register on the worker, as a shorthand for calling `register_workflow` on each or `register_workflows` on all of them.

        :param lifespan: A lifespan function to run on the worker. This function will be called when the worker is started, and can be used to perform any setup or teardown tasks.

        :returns: The created `Worker` object, which exposes an instance method `start` which can be called to start the worker.
        """

        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = None

        return Worker(
            name=name,
            slots=slots,
            durable_slots=durable_slots,
            labels=labels,
            config=self._client.config,
            debug=self._client.debug,
            owned_loop=loop is None,
            workflows=workflows,
            lifespan=lifespan,
        )

    @overload
    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
        default_filters: list[DefaultFilter] | None = None,
    ) -> Workflow[EmptyModel]: ...

    @overload
    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: type[TWorkflowInput],
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
        default_filters: list[DefaultFilter] | None = None,
    ) -> Workflow[TWorkflowInput]: ...

    def workflow(
        self,
        *,
        name: str,
        description: str | None = None,
        input_validator: type[TWorkflowInput] | None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        task_defaults: TaskDefaults = TaskDefaults(),
        default_filters: list[DefaultFilter] | None = None,
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
        """
        Define a Hatchet workflow, which can then declare `task`s and be `run`, `schedule`d, and so on.

        :param name: The name of the workflow.

        :param description: A description for the workflow

        :param input_validator: A Pydantic model to use as a validator for the `input` to the tasks in the workflow. If no validator is provided, defaults to an `EmptyModel` under the hood. The `EmptyModel` is a Pydantic model with no fields specified, and with the `extra` config option set to `"allow"`.

        :param on_events: A list of event triggers for the workflow - events which cause the workflow to be run.

        :param on_crons: A list of cron triggers for the workflow.

        :param version: A version for the workflow

        :param sticky: A sticky strategy for the workflow

        :param default_priority: The priority of the workflow. Higher values will cause this workflow to have priority in scheduling over other, lower priority ones.

        :param concurrency: A concurrency object controlling the concurrency settings for this workflow.

        :param task_defaults: A `TaskDefaults` object controlling the default task settings for this workflow.

        :param default_filters: A list of filters to create with the workflow is created. Note that this is a helper to allow you to create filters "declaratively" without needing to make a separate API call once the workflow is created to create them.

        :returns: The created `Workflow` object, which can be used to declare tasks, run the workflow, and so on.
        """

        return Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                version=version,
                description=description,
                on_events=on_events or [],
                on_crons=on_crons or [],
                sticky=sticky,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(type[TWorkflowInput], EmptyModel),
                task_defaults=task_defaults,
                default_priority=default_priority,
                default_filters=default_filters or [],
            ),
            self,
        )

    @overload
    def task(
        self,
        *,
        name: str | None = None,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> Callable[
        [Callable[[EmptyModel, Context], R | CoroutineLike[R]]],
        Standalone[EmptyModel, R],
    ]: ...

    @overload
    def task(
        self,
        *,
        name: str | None = None,
        description: str | None = None,
        input_validator: type[TWorkflowInput],
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> Callable[
        [Callable[[TWorkflowInput, Context], R | CoroutineLike[R]]],
        Standalone[TWorkflowInput, R],
    ]: ...

    def task(
        self,
        *,
        name: str | None = None,
        description: str | None = None,
        input_validator: type[TWorkflowInput] | None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> (
        Callable[
            [Callable[[EmptyModel, Context], R | CoroutineLike[R]]],
            Standalone[EmptyModel, R],
        ]
        | Callable[
            [Callable[[TWorkflowInput, Context], R | CoroutineLike[R]]],
            Standalone[TWorkflowInput, R],
        ]
    ):
        """
        A decorator to transform a function into a standalone Hatchet task that runs as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.

        :param description: An optional description for the task.

        :param input_validator: A Pydantic model to use as a validator for the input to the task. If no validator is provided, defaults to an `EmptyModel`.

        :param on_events: A list of event triggers for the task - events which cause the task to be run.

        :param on_crons: A list of cron triggers for the task.

        :param version: A version for the task.

        :param sticky: A sticky strategy for the task.

        :param default_priority: The priority of the task. Higher values will cause this task to have priority in scheduling.

        :param concurrency: A concurrency object controlling the concurrency settings for this task.

        :param schedule_timeout: The maximum time allowed for scheduling the task.

        :param execution_timeout: The maximum time allowed for executing the task.

        :param retries: The number of times to retry the task before failing.

        :param rate_limits: A list of rate limit configurations for the task.

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param default_filters: A list of filters to create with the task is created. Note that this is a helper to allow you to create filters "declaratively" without needing to make a separate API call once the task is created to create them.

        :returns: A decorator which creates a `Standalone` task object.
        """

        def inner(
            func: Callable[[TWorkflowInput, Context], R | CoroutineLike[R]],
        ) -> Standalone[TWorkflowInput, R]:
            inferred_name = name or func.__name__

            workflow = Workflow[TWorkflowInput](
                WorkflowConfig(
                    name=inferred_name,
                    version=version,
                    description=description,
                    on_events=on_events or [],
                    on_crons=on_crons or [],
                    sticky=sticky,
                    default_priority=default_priority,
                    input_validator=input_validator
                    or cast(type[TWorkflowInput], EmptyModel),
                    default_filters=default_filters or [],
                ),
                self,
            )

            if isinstance(concurrency, list):
                _concurrency = concurrency
            elif isinstance(concurrency, ConcurrencyExpression):
                _concurrency = [concurrency]
            else:
                _concurrency = []

            task_wrapper = workflow.task(
                name=inferred_name,
                schedule_timeout=schedule_timeout,
                execution_timeout=execution_timeout,
                parents=[],
                retries=retries,
                rate_limits=rate_limits or [],
                desired_worker_labels=desired_worker_labels or {},
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=_concurrency,
            )

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
        name: str | None = None,
        description: str | None = None,
        input_validator: None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> Callable[
        [Callable[[EmptyModel, DurableContext], R | CoroutineLike[R]]],
        Standalone[EmptyModel, R],
    ]: ...

    @overload
    def durable_task(
        self,
        *,
        name: str | None = None,
        description: str | None = None,
        input_validator: type[TWorkflowInput],
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> Callable[
        [Callable[[TWorkflowInput, DurableContext], R | CoroutineLike[R]]],
        Standalone[TWorkflowInput, R],
    ]: ...

    def durable_task(
        self,
        *,
        name: str | None = None,
        description: str | None = None,
        input_validator: type[TWorkflowInput] | None = None,
        on_events: list[str] | None = None,
        on_crons: list[str] | None = None,
        version: str | None = None,
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | list[ConcurrencyExpression] | None = None,
        schedule_timeout: Duration = timedelta(minutes=5),
        execution_timeout: Duration = timedelta(seconds=60),
        retries: int = 0,
        rate_limits: list[RateLimit] | None = None,
        desired_worker_labels: dict[str, DesiredWorkerLabel] | None = None,
        backoff_factor: float | None = None,
        backoff_max_seconds: int | None = None,
        default_filters: list[DefaultFilter] | None = None,
    ) -> (
        Callable[
            [Callable[[EmptyModel, DurableContext], R | CoroutineLike[R]]],
            Standalone[EmptyModel, R],
        ]
        | Callable[
            [Callable[[TWorkflowInput, DurableContext], R | CoroutineLike[R]]],
            Standalone[TWorkflowInput, R],
        ]
    ):
        """
        A decorator to transform a function into a standalone Hatchet _durable_ task that runs as part of a workflow.

        :param name: The name of the task. If not specified, defaults to the name of the function being wrapped by the `task` decorator.

        :param description: An optional description for the task.

        :param input_validator: A Pydantic model to use as a validator for the input to the task. If no validator is provided, defaults to an `EmptyModel`.

        :param on_events: A list of event triggers for the task - events which cause the task to be run.

        :param on_crons: A list of cron triggers for the task.

        :param version: A version for the task.

        :param sticky: A sticky strategy for the task.

        :param default_priority: The priority of the task. Higher values will cause this task to have priority in scheduling.

        :param concurrency: A concurrency object controlling the concurrency settings for this task.

        :param schedule_timeout: The maximum time allowed for scheduling the task.

        :param execution_timeout: The maximum time allowed for executing the task.

        :param retries: The number of times to retry the task before failing.

        :param rate_limits: A list of rate limit configurations for the task.

        :param desired_worker_labels: A dictionary of desired worker labels that determine to which worker the task should be assigned.

        :param backoff_factor: The backoff factor for controlling exponential backoff in retries.

        :param backoff_max_seconds: The maximum number of seconds to allow retries with exponential backoff to continue.

        :param default_filters: A list of filters to create with the task is created. Note that this is a helper to allow you to create filters "declaratively" without needing to make a separate API call once the task is created to create them.

        :returns: A decorator which creates a `Standalone` task object.
        """

        def inner(
            func: Callable[[TWorkflowInput, DurableContext], R | CoroutineLike[R]],
        ) -> Standalone[TWorkflowInput, R]:
            inferred_name = name or func.__name__
            workflow = Workflow[TWorkflowInput](
                WorkflowConfig(
                    name=inferred_name,
                    version=version,
                    description=description,
                    on_events=on_events or [],
                    on_crons=on_crons or [],
                    sticky=sticky,
                    input_validator=input_validator
                    or cast(type[TWorkflowInput], EmptyModel),
                    default_priority=default_priority,
                    default_filters=default_filters or [],
                ),
                self,
            )

            if isinstance(concurrency, list):
                _concurrency = concurrency
            elif isinstance(concurrency, ConcurrencyExpression):
                _concurrency = [concurrency]
            else:
                _concurrency = []

            task_wrapper = workflow.durable_task(
                name=inferred_name,
                schedule_timeout=schedule_timeout,
                execution_timeout=execution_timeout,
                parents=[],
                retries=retries,
                rate_limits=rate_limits or [],
                desired_worker_labels=desired_worker_labels or {},
                backoff_factor=backoff_factor,
                backoff_max_seconds=backoff_max_seconds,
                concurrency=_concurrency,
            )

            created_task = task_wrapper(func)

            return Standalone[TWorkflowInput, R](
                workflow=workflow,
                task=created_task,
            )

        return inner
