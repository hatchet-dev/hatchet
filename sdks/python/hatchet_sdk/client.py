import asyncio
from typing import Callable

import grpc

from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient, new_event
from hatchet_sdk.clients.rest_client import RestApi
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.connection import new_conn
from hatchet_sdk.loader import ClientConfig


class Client:
    @classmethod
    def from_environment(
        cls,
        defaults: ClientConfig = ClientConfig(),
        debug: bool = False,
        *opts_functions: Callable[[ClientConfig], None],
    ) -> "Client":
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        for opt_function in opts_functions:
            opt_function(defaults)

        return cls.from_config(defaults, debug)

    @classmethod
    def from_config(
        cls,
        config: ClientConfig = ClientConfig(),
        debug: bool = False,
    ) -> "Client":
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        if config.tls_config is None:
            raise ValueError("TLS config is required")

        if config.host_port is None:
            raise ValueError("Host and port are required")

        conn: grpc.Channel = new_conn(config, False)

        # Instantiate clients
        event_client = new_event(conn, config)
        admin_client = AdminClient(config)
        dispatcher_client = DispatcherClient(config)
        rest_client = RestApi(config.server_url, config.token, config.tenant_id)
        workflow_listener = None  # Initialize this if needed

        return cls(
            event_client,
            admin_client,
            dispatcher_client,
            workflow_listener,
            rest_client,
            config,
            debug,
        )

    def __init__(
        self,
        event_client: EventClient,
        admin_client: AdminClient,
        dispatcher_client: DispatcherClient,
        workflow_listener: PooledWorkflowRunListener | None,
        rest_client: RestApi,
        config: ClientConfig,
        debug: bool = False,
    ):
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        self.admin = admin_client
        self.dispatcher = dispatcher_client
        self.event = event_client
        self.rest = rest_client
        self.config = config
        self.listener = RunEventListenerClient(config)
        self.workflow_listener = workflow_listener
        self.logInterceptor = config.logger
        self.debug = debug


new_client = Client.from_environment
new_client_raw = Client.from_config
