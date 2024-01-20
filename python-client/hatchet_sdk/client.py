# relative imports
from .clients.admin import AdminClientImpl, new_admin
from .clients.events import EventClientImpl, new_event
from .clients.dispatcher import DispatcherClientImpl, new_dispatcher

from .loader import ConfigLoader, ClientConfig
import grpc

class Client:
    def admin(self):
        raise NotImplementedError

    def dispatcher(self):
        raise NotImplementedError

    def event(self):
        raise NotImplementedError
        
class ClientImpl(Client):
    def __init__(self, event_client, admin_client : AdminClientImpl, dispatcher_client):
        # self.conn = conn
        # self.tenant_id = tenant_id
        # self.logger = logger
        # self.validator = validator
        self.admin = admin_client
        self.dispatcher = dispatcher_client
        self.event = event_client

    def admin(self) -> AdminClientImpl:
        return self.admin

    def dispatcher(self) -> DispatcherClientImpl:
        return self.dispatcher

    def event(self) -> EventClientImpl:
        return self.event
    
def with_host_port(host: str, port: int):
    def with_host_port_impl(config: ClientConfig):
        config.host = host
        config.port = port

    return with_host_port_impl

def new_client(*opts_functions):
    config : ClientConfig = ConfigLoader(".").load_client_config()

    for opt_function in opts_functions:
        opt_function(config)

    if config.tls_config is None:
        raise ValueError("TLS config is required")
    
    if config.host_port is None:
        raise ValueError("Host and port are required")
        
    root = open(config.tls_config.ca_file, "rb").read()
    private_key = open(config.tls_config.key_file, "rb").read()
    certificate_chain = open(config.tls_config.cert_file, "rb").read()

    conn = grpc.secure_channel(
        target=config.host_port,
        credentials=grpc.ssl_channel_credentials(root_certificates=root, private_key=private_key, certificate_chain=certificate_chain),
        options=[('grpc.ssl_target_name_override', config.tls_config.server_name)],
    )

    # Instantiate client implementations
    event_client = new_event(conn, config)
    admin_client = new_admin(conn, config)
    dispatcher_client = new_dispatcher(conn, config)

    return ClientImpl(event_client, admin_client, dispatcher_client)