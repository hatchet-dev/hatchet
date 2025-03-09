import asyncio
import logging
from typing import Any, Type, TypeVar, cast, overload

from hatchet_sdk.client import Client, new_client, new_client_raw
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

    @classmethod
    def from_environment(
        cls, defaults: ClientConfig = ClientConfig(), **kwargs: Any
    ) -> "Hatchet":
        return cls(client=new_client(defaults), **kwargs)

    @classmethod
    def from_config(cls, config: ClientConfig, **kwargs: Any) -> "Hatchet":
        return cls(client=new_client_raw(config), **kwargs)

    def __init__(
        self,
        debug: bool = False,
        client: Client | None = None,
        config: ClientConfig = ClientConfig(),
    ):
        """
        Initialize a new Hatchet instance.

        Args:
            debug (bool, optional): Enable debug logging. Defaults to False.
            client (Optional[Client], optional): A pre-configured Client instance. Defaults to None.
            config (ClientConfig, optional): Configuration for creating a new Client. Defaults to ClientConfig().
        """

        if debug:
            logger.setLevel(logging.DEBUG)

        self._client = client if client else new_client(config, debug)
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
        max_runs: int | None = None,
        labels: dict[str, str | int] = {},
        workflows: list[Workflow[Any]] = [],
    ) -> Worker:
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = None

        return Worker(
            name=name,
            max_runs=max_runs,
            labels=labels,
            config=self._client.config,
            debug=self._client.debug,
            owned_loop=loop is None,
        )

    @overload
    def workflow(
        self,
        *,
        name: str,
        input_validator: None = None,
        on_events: list[str] = [],
        on_crons: list[str] = [],
        version: str = "",
        timeout: str = "60m",
        schedule_timeout: str = "5m",
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
        version: str = "",
        timeout: str = "60m",
        schedule_timeout: str = "5m",
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
        version: str = "",
        timeout: str = "60m",
        schedule_timeout: str = "5m",
        sticky: StickyStrategy | None = None,
        default_priority: int = 1,
        concurrency: ConcurrencyExpression | None = None,
    ) -> Workflow[EmptyModel] | Workflow[TWorkflowInput]:
        return Workflow[TWorkflowInput](
            WorkflowConfig(
                name=name,
                on_events=on_events,
                on_crons=on_crons,
                version=version,
                timeout=timeout,
                schedule_timeout=schedule_timeout,
                sticky=sticky,
                default_priority=default_priority,
                concurrency=concurrency,
                input_validator=input_validator
                or cast(Type[TWorkflowInput], EmptyModel),
            ),
            self,
        )
