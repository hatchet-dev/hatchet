import os
from logging import Logger, getLogger
from typing import Dict, Optional
from warnings import warn

import yaml

from .token import get_addresses_from_jwt, get_tenant_id_from_jwt


class ClientTLSConfig:
    def __init__(
        self,
        tls_strategy: str,
        cert_file: str,
        key_file: str,
        ca_file: str,
        server_name: str,
    ):
        self.tls_strategy = tls_strategy
        self.cert_file = cert_file
        self.key_file = key_file
        self.ca_file = ca_file
        self.server_name = server_name


class ClientConfig:
    logInterceptor: Logger

    def __init__(
        self,
        tenant_id: str = None,
        tls_config: ClientTLSConfig = None,
        token: str = None,
        host_port: str = "localhost:7070",
        server_url: str = "https://app.dev.hatchet-tools.com",
        namespace: str = None,
        listener_v2_timeout: int = None,
        logger: Logger = None,
        grpc_max_recv_message_length: int = 4 * 1024 * 1024,  # 4MB
        grpc_max_send_message_length: int = 4 * 1024 * 1024,  # 4MB
        worker_healthcheck_port: int | None = None,
        worker_healthcheck_enabled: bool | None = None,
        worker_preset_labels: dict[str, str] = {},
        enable_force_kill_sync_threads: bool = False,
    ):
        self.tenant_id = tenant_id
        self.tls_config = tls_config
        self.host_port = host_port
        self.token = token
        self.server_url = server_url
        self.namespace = ""
        self.logInterceptor = logger
        self.grpc_max_recv_message_length = grpc_max_recv_message_length
        self.grpc_max_send_message_length = grpc_max_send_message_length
        self.worker_healthcheck_port = worker_healthcheck_port
        self.worker_healthcheck_enabled = worker_healthcheck_enabled
        self.worker_preset_labels = worker_preset_labels
        self.enable_force_kill_sync_threads = enable_force_kill_sync_threads

        if not self.logInterceptor:
            self.logInterceptor = getLogger()

        # case on whether the namespace already has a trailing underscore
        if namespace and not namespace.endswith("_"):
            self.namespace = f"{namespace}_"
        elif namespace:
            self.namespace = namespace

        self.namespace = self.namespace.lower()

        self.listener_v2_timeout = listener_v2_timeout


class ConfigLoader:
    def __init__(self, directory: str):
        self.directory = directory

    def load_client_config(self, defaults: ClientConfig) -> ClientConfig:
        config_file_path = os.path.join(self.directory, "client.yaml")
        config_data: object = {"tls": {}}

        # determine if client.yaml exists
        if os.path.exists(config_file_path):
            with open(config_file_path, "r") as file:
                config_data = yaml.safe_load(file)

        def get_config_value(key, env_var):
            if key in config_data:
                return config_data[key]

            if self._get_env_var(env_var) is not None:
                return self._get_env_var(env_var)

            return getattr(defaults, key, None)

        namespace = get_config_value("namespace", "HATCHET_CLIENT_NAMESPACE")

        tenant_id = get_config_value("tenantId", "HATCHET_CLIENT_TENANT_ID")
        token = get_config_value("token", "HATCHET_CLIENT_TOKEN")
        listener_v2_timeout = get_config_value(
            "listener_v2_timeout", "HATCHET_CLIENT_LISTENER_V2_TIMEOUT"
        )
        listener_v2_timeout = int(listener_v2_timeout) if listener_v2_timeout else None

        if not token:
            raise ValueError(
                "Token must be set via HATCHET_CLIENT_TOKEN environment variable"
            )

        host_port = get_config_value("hostPort", "HATCHET_CLIENT_HOST_PORT")
        server_url: str | None = None

        grpc_max_recv_message_length = get_config_value(
            "grpc_max_recv_message_length",
            "HATCHET_CLIENT_GRPC_MAX_RECV_MESSAGE_LENGTH",
        )
        grpc_max_send_message_length = get_config_value(
            "grpc_max_send_message_length",
            "HATCHET_CLIENT_GRPC_MAX_SEND_MESSAGE_LENGTH",
        )

        if grpc_max_recv_message_length:
            grpc_max_recv_message_length = int(grpc_max_recv_message_length)

        if grpc_max_send_message_length:
            grpc_max_send_message_length = int(grpc_max_send_message_length)

        if not host_port:
            # extract host and port from token
            server_url, grpc_broadcast_address = get_addresses_from_jwt(token)
            host_port = grpc_broadcast_address

        if not tenant_id:
            tenant_id = get_tenant_id_from_jwt(token)

        tls_config = self._load_tls_config(config_data["tls"], host_port)

        worker_healthcheck_port = int(
            get_config_value(
                "worker_healthcheck_port", "HATCHET_CLIENT_WORKER_HEALTHCHECK_PORT"
            )
            or 8001
        )

        worker_healthcheck_enabled = (
            str(
                get_config_value(
                    "worker_healthcheck_port",
                    "HATCHET_CLIENT_WORKER_HEALTHCHECK_ENABLED",
                )
            )
            == "True"
        )

        #  Add preset labels to the worker config
        worker_preset_labels: dict[str, str] = defaults.worker_preset_labels

        autoscaling_target = get_config_value(
            "autoscaling_target", "HATCHET_CLIENT_AUTOSCALING_TARGET"
        )

        if autoscaling_target:
            worker_preset_labels["hatchet-autoscaling-target"] = autoscaling_target

        legacy_otlp_headers = get_config_value(
            "otel_exporter_otlp_endpoint", "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_ENDPOINT"
        )

        legacy_otlp_headers = get_config_value(
            "otel_exporter_otlp_headers", "HATCHET_CLIENT_OTEL_EXPORTER_OTLP_HEADERS"
        )

        if legacy_otlp_headers or legacy_otlp_headers:
            warn(
                "The `otel_exporter_otlp_*` fields are no longer supported as of SDK version `0.46.0`. Please see the documentation on OpenTelemetry at https://docs.hatchet.run/home/features/opentelemetry for more information on how to migrate to the new `HatchetInstrumentor`."
            )

        enable_force_kill_sync_threads = bool(
            get_config_value(
                "enable_force_kill_sync_threads",
                "HATCHET_CLIENT_ENABLE_FORCE_KILL_SYNC_THREADS",
            )
            == "True"
            or False
        )
        return ClientConfig(
            tenant_id=tenant_id,
            tls_config=tls_config,
            token=token,
            host_port=host_port,
            server_url=server_url,
            namespace=namespace,
            listener_v2_timeout=listener_v2_timeout,
            logger=defaults.logInterceptor,
            grpc_max_recv_message_length=grpc_max_recv_message_length,
            grpc_max_send_message_length=grpc_max_send_message_length,
            worker_healthcheck_port=worker_healthcheck_port,
            worker_healthcheck_enabled=worker_healthcheck_enabled,
            worker_preset_labels=worker_preset_labels,
            enable_force_kill_sync_threads=enable_force_kill_sync_threads,
        )

    def _load_tls_config(self, tls_data: Dict, host_port) -> ClientTLSConfig:
        tls_strategy = (
            tls_data["tlsStrategy"]
            if "tlsStrategy" in tls_data
            else self._get_env_var("HATCHET_CLIENT_TLS_STRATEGY")
        )

        if not tls_strategy:
            tls_strategy = "tls"

        cert_file = (
            tls_data["tlsCertFile"]
            if "tlsCertFile" in tls_data
            else self._get_env_var("HATCHET_CLIENT_TLS_CERT_FILE")
        )
        key_file = (
            tls_data["tlsKeyFile"]
            if "tlsKeyFile" in tls_data
            else self._get_env_var("HATCHET_CLIENT_TLS_KEY_FILE")
        )
        ca_file = (
            tls_data["tlsRootCAFile"]
            if "tlsRootCAFile" in tls_data
            else self._get_env_var("HATCHET_CLIENT_TLS_ROOT_CA_FILE")
        )

        server_name = (
            tls_data["tlsServerName"]
            if "tlsServerName" in tls_data
            else self._get_env_var("HATCHET_CLIENT_TLS_SERVER_NAME")
        )

        # if server_name is not set, use the host from the host_port
        if not server_name:
            server_name = host_port.split(":")[0]

        return ClientTLSConfig(tls_strategy, cert_file, key_file, ca_file, server_name)

    @staticmethod
    def _get_env_var(env_var: str, default: Optional[str] = None) -> str:
        return os.environ.get(env_var, default)
