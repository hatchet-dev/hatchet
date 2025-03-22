import asyncio

import grpc

from hatchet_sdk.clients.admin import AdminClient
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.events import EventClient, new_event
from hatchet_sdk.clients.rest_client import RestApi
from hatchet_sdk.clients.run_event_listener import RunEventListenerClient
from hatchet_sdk.clients.workflow_listener import PooledWorkflowRunListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn


class Client:
    def __init__(
        self,
        config: ClientConfig,
        event_client: EventClient | None = None,
        admin_client: AdminClient | None = None,
        dispatcher_client: DispatcherClient | None = None,
        workflow_listener: PooledWorkflowRunListener | None | None = None,
        rest_client: RestApi | None = None,
        debug: bool = False,
    ):
        try:
            loop = asyncio.get_running_loop()
        except RuntimeError:
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)

        conn: grpc.Channel = new_conn(config, False)

        self.config = config
        self.admin = admin_client or AdminClient(config)
        self.dispatcher = dispatcher_client or DispatcherClient(config)
        self.event = event_client or new_event(conn, config)
        self.rest = rest_client or RestApi(
            config.server_url, config.token, config.tenant_id
        )
        self.listener = RunEventListenerClient(config)
        self.workflow_listener = workflow_listener
        self.logInterceptor = config.logger
        self.debug = debug
