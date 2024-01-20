import os
import yaml
from typing import Any, Optional, Dict

class ClientTLSConfig:
    def __init__(self, cert_file: str, key_file: str, ca_file: str, server_name: str):
        self.cert_file = cert_file
        self.key_file = key_file
        self.ca_file = ca_file
        self.server_name = server_name

class ClientConfig:
    def __init__(self, tenant_id: str, tls_config: ClientTLSConfig, host_port: str="localhost:7070"):
        self.tenant_id = tenant_id
        self.tls_config = tls_config
        self.host_port = host_port

class ConfigLoader:
    def __init__(self, directory: str):
        self.directory = directory

    def load_client_config(self) -> ClientConfig:
        config_file_path = os.path.join(self.directory, "client.yaml")

        config_data : Any = {
            "tls": {},
        }

        # determine if client.yaml exists
        if os.path.exists(config_file_path):
            with open(config_file_path, 'r') as file:
                config_data = yaml.safe_load(file)

        tls_config = self._load_tls_config(config_data['tls'])
        tenant_id = config_data['tenantId'] if 'tenantId' in config_data else self._get_env_var('HATCHET_CLIENT_TENANT_ID')
        host_port = config_data['hostPort'] if 'hostPort' in config_data else self._get_env_var('HATCHET_CLIENT_HOST_PORT')

        return ClientConfig(tenant_id, tls_config, host_port)

    def _load_tls_config(self, tls_data: Dict) -> ClientTLSConfig:
        cert_file = tls_data['tlsCertFile'] if 'tlsCertFile' in tls_data else self._get_env_var('HATCHET_CLIENT_TLS_CERT_FILE')
        key_file = tls_data['tlsKeyFile'] if 'tlsKeyFile' in tls_data else self._get_env_var('HATCHET_CLIENT_TLS_KEY_FILE')
        ca_file = tls_data['tlsRootCAFile'] if 'tlsRootCAFile' in tls_data else self._get_env_var('HATCHET_CLIENT_TLS_ROOT_CA_FILE')
        server_name = tls_data['tlsServerName'] if 'tlsServerName' in tls_data else self._get_env_var('HATCHET_CLIENT_TLS_SERVER_NAME')

        return ClientTLSConfig(cert_file, key_file, ca_file, server_name)

    @staticmethod
    def _get_env_var(env_var: str, default: Optional[str] = None) -> str:
        return os.environ.get(env_var, default)
