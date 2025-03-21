import asyncio
import logging
from datetime import timedelta
from typing import Any, Type, TypeVar, cast, overload

from hatchet_sdk.client import Client
from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient
from hatchet_sdk.clients.rest_client import RestApi
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.features.cron import CronClient
from hatchet_sdk.features.scheduled import ScheduledClient
from hatchet_sdk.logger import logger
from hatchet_sdk.runnables.types import (
    ConcurrencyExpression,
    EmptyModel,
    StickyStrategy,
    TWorkflowInput,
    WorkflowConfig,
)
from hatchet_sdk.runnables.workflow import Workflow
from hatchet_sdk.worker.worker import Worker

R = TypeVar("R")


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

    _client: Client
    cron: CronClient
    scheduled: ScheduledClient

    def __init__(
        self,
        debug: bool = False,
        client: Client | None = None,
        config: ClientConfig = ClientConfig(),
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

        self._client = client if client else Client(config=config, debug=debug)
        self.cron = CronClient(self._client)
        self.scheduled = ScheduledClient(self._client)

    @property
    def admin(self) -> AdminClient:
        return self._client.admin

    @property
    def dispatcher(self) -> DispatcherClient:
        return self._client.dispatcher

    @property
    def event(self) -> EventClient:
        return self._client.event

    @property
    def rest(self) -> RestApi:
        return self._client.rest

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
        workflows: list[Workflow[Any]] = [],
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
        input_validator: None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        schedule_timeout: timedelta | str = timedelta(minutes=5),
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
    ) -> Workflow[EmptyModel]: ...

    @overload
    def workflow(
        self,
        *,
        name: str,
        input_validator: Type[TWorkflowInput],
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        schedule_timeout: timedelta | str = timedelta(minutes=5),
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
    ) -> Workflow[TWorkflowInput]: ...

    def workflow(
        self,
        *,
        name: str,
        input_validator: Type[TWorkflowInput] | None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str | None = None,
        schedule_timeout: timedelta | str = timedelta(minutes=5),
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
        """
        Define a Hatchet workflow, which can then declare `task`s and be `run`, `schedule`d, and so on.

        :param name: The name of the workflow.
        :type name: str

        :param input_validator: A Pydantic model to use as a validator for the `input` to the tasks in the workflow. If no validator is provided, defaults to an `EmptyModel` under the hood. The `EmptyModel` is a Pydantic model with no fields specified, and with the `extra` config option set to `"allow"`.
        :type input_validator: Type[BaseModel]

        :param on_events: A list of event triggers for the workflow - events which cause the workflow to be run. Defaults to an empty list, meaning the workflow will not be run on any event pushes.
        :type on_events: list[str]

        :param on_crons: A list of cron triggers for the workflow. Defaults to an empty list, meaning the workflow will not be run on any cron schedules.
        :type on_crons: list[str]

        :param version: A version for the workflow. Default: None
        :type version: str | None

        :param schedule_timeout: The maximum amount of time that a workflow run that has been queued (scheduled) can wait before beginning to execute. For instance, setting `timedelta(minutes=5)` will cancel the workflow run if it does not start after five minutes. Default: five minutes.
        :type schedule_timeout: `datetime.timedelta`

        :param sticky: A sticky strategy for the workflow. Default: `None`
        :type sticky: StickyStategy

        :param default_priority: The priority of the workflow. Higher values will cause this workflow to have priority in scheduling over other, lower priority ones. Default: `1`
        :type default_priority: int

        :param concurrency: A concurrency object controlling the concurrency settings for this workflow.
        :type concurrency: ConcurrencyExpression | None

        :returns: The created `Workflow` object, which can be used to declare tasks, run the workflow, and so on.
        :rtype: Workflow
        """

        return Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                on_events=on_events,
                on_crons=on_crons,
                version=version or "",
                schedule_timeout=schedule_timeout,
                sticky=sticky,
                default_priority=default_priority,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(Type[TWorkflowInput], EmptyModel),
            ),
            self,
        )
