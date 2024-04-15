# relative imports
import os
from typing import Any

from hatchet_sdk.connection import new_conn
from .clients.admin import AdminClientImpl, new_admin
from .clients.events import EventClientImpl, new_event
from .clients.dispatcher import DispatcherClientImpl, new_dispatcher
from .clients.listener import ListenerClientImpl, new_listener

from .loader import ConfigLoader, ClientConfig
import grpc

from .clients.rest.api_client import ApiClient
from .clients.rest.api.workflow_api import WorkflowApi
from .clients.rest.api.workflow_run_api import WorkflowRunApi
from .clients.rest.configuration import Configuration
from .clients.rest_client import RestApi

class Client:
    admin: AdminClientImpl
    dispatcher: DispatcherClientImpl
    event: EventClientImpl
    listener: ListenerClientImpl
    rest: RestApi


class ClientImpl(Client):
    def __init__(
            self,
            event_client: EventClientImpl,
            admin_client: AdminClientImpl,
            dispatcher_client: DispatcherClientImpl,
            listener_client: ListenerClientImpl,
            rest_client: RestApi,
            config: ClientConfig
        ):
        self.admin = admin_client
        self.dispatcher = dispatcher_client
        self.event = event_client
        self.listener = listener_client
        self.rest = rest_client
        self.config = config

def with_host_port(host: str, port: int):
    def with_host_port_impl(config: ClientConfig):
        config.host = host
        config.port = port

    return with_host_port_impl


def new_client(*opts_functions):
    config: ClientConfig = ConfigLoader(".").load_client_config()

    for opt_function in opts_functions:
        opt_function(config)

    if config.tls_config is None:
        raise ValueError("TLS config is required")

    if config.host_port is None:
        raise ValueError("Host and port are required")

    conn : grpc.Channel = new_conn(config)

    # Instantiate client implementations
    event_client = new_event(conn, config)
    admin_client = new_admin(conn, config)
    dispatcher_client = new_dispatcher(conn, config)
    listener_client = new_listener(conn, config)
    rest_client = RestApi(config.server_url, config.token, config.tenant_id)

    return ClientImpl(event_client, admin_client, dispatcher_client, listener_client, rest_client, config)
