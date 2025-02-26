import asyncio
from logging import Logger
from typing import Callable

import grpc

from hatchet_sdk.v0.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.v0.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.v0.connection import new_conn

from .clients.admin import AdminClient, new_admin
from .clients.dispatcher.dispatcher import DispatcherClient, new_dispatcher
from .clients.events import EventClient, new_event
from .clients.rest_client import RestApi
from .loader import ClientConfig, ConfigLoader


class Client:
    admin: AdminClient
    dispatcher: DispatcherClient
    event: EventClient
    rest: RestApi
    workflow_listener: PooledWorkflowRunListener
    logInterceptor: Logger
    debug: bool = False

    @classmethod
    def from_environment(
        cls,
        defaults: ClientConfig = ClientConfig(),
        debug: bool = False,
        *opts_functions: Callable[[ClientConfig], None],
    ):
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        config: ClientConfig = ConfigLoader(".").load_client_config(defaults)
        for opt_function in opts_functions:
            opt_function(config)

        return cls.from_config(config, debug)

    @classmethod
    def from_config(
        cls,
        config: ClientConfig = ClientConfig(),
        debug: bool = False,
    ):
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        if config.tls_config is None:
            raise ValueError("TLS config is required")

        if config.host_port is None:
            raise ValueError("Host and port are required")

        conn: grpc.Channel = new_conn(config)

        # Instantiate clients
        event_client = new_event(conn, config)
        admin_client = new_admin(config)
        dispatcher_client = new_dispatcher(config)
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
        workflow_listener: PooledWorkflowRunListener,
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
        self.logInterceptor = config.logInterceptor
        self.debug = debug


def with_host_port(host: str, port: int):
    def with_host_port_impl(config: ClientConfig):
        config.host = host
        config.port = port

    return with_host_port_impl


new_client = Client.from_environment
new_client_raw = Client.from_config
