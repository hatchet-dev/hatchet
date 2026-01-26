import json
from datetime import timedelta
from logging import Logger, getLogger
from typing import overload

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

from hatchet_sdk.token import get_addresses_from_jwt, get_tenant_id_from_jwt
from hatchet_sdk.utils.opentelemetry import OTelAttribute


def create_settings_config(env_prefix: str) -> SettingsConfigDict:
    return SettingsConfigDict(
        env_prefix=env_prefix,
        env_file=(".env", ".env.hatchet", ".env.dev", ".env.local"),
        extra="ignore",
    )


class ClientTLSConfig(BaseSettings):
    model_config = create_settings_config(
        env_prefix="HATCHET_CLIENT_TLS_",
    )

    strategy: str = "tls"
    cert_file: str | None = None
    key_file: str | None = None
    root_ca_file: str | None = None
    server_name: str = ""


class HealthcheckConfig(BaseSettings):
    model_config = create_settings_config(
        env_prefix="HATCHET_CLIENT_WORKER_HEALTHCHECK_",
    )

    port: int = 8001
    enabled: bool = False
    event_loop_block_threshold_seconds: timedelta = Field(
        default=timedelta(seconds=5),
        description="If the worker listener process event loop appears blocked longer than this threshold, /health returns 503. Value is interpreted as seconds.",
    )
    bind_address: str | None = "0.0.0.0"

    @field_validator("event_loop_block_threshold_seconds", mode="before")
    @classmethod
    def validate_event_loop_block_threshold_seconds(
        cls, value: timedelta | int | float | str
    ) -> timedelta:
        # Settings env vars are strings; interpret as seconds.
        if isinstance(value, timedelta):
            return value

        if isinstance(value, int | float):
            return timedelta(seconds=float(value))

        v = value.strip()
        # Allow a small convenience suffix, but keep "seconds" as the contract.
        if v.endswith("s"):
            v = v[:-1].strip()

        return timedelta(seconds=float(v))

    @field_validator("bind_address", mode="after")
    @classmethod
    def validate_bind_address(cls, value: str | None) -> str | None:
        if value is None:
            return None

        if value.lower() == "none" or not value.strip():
            return None

        return value


class OpenTelemetryConfig(BaseSettings):
    model_config = create_settings_config(
        env_prefix="HATCHET_CLIENT_OPENTELEMETRY_",
    )

    excluded_attributes: list[OTelAttribute] = Field(
        default_factory=list,
        description='Note that if specifying this field via an environment variable, the variable must be a valid JSON array. For example: \'["action_name", "action_payload"]\'',
    )

    include_task_name_in_start_step_run_span_name: bool = False


class TenacityConfig(BaseSettings):
    model_config = create_settings_config(
        env_prefix="HATCHET_CLIENT_TENACITY_",
    )

    max_attempts: int = 5


DEFAULT_HOST_PORT = "localhost:7070"


class ClientConfig(BaseSettings):
    model_config = create_settings_config(
        env_prefix="HATCHET_CLIENT_",
    )

    token: str = ""
    logger: Logger = getLogger()

    tenant_id: str = ""
    host_port: str = DEFAULT_HOST_PORT
    server_url: str = "https://app.dev.hatchet-tools.com"
    namespace: str = ""

    tls_config: ClientTLSConfig = Field(default_factory=lambda: ClientTLSConfig())
    healthcheck: HealthcheckConfig = Field(default_factory=lambda: HealthcheckConfig())
    otel: OpenTelemetryConfig = Field(default_factory=lambda: OpenTelemetryConfig())

    listener_v2_timeout: int | None = None
    grpc_max_recv_message_length: int = Field(
        default=4 * 1024 * 1024, description="4MB default"
    )
    grpc_max_send_message_length: int = Field(
        default=4 * 1024 * 1024, description="4MB default"
    )

    worker_preset_labels: dict[str, str] = Field(default_factory=dict)

    enable_force_kill_sync_threads: bool = False
    enable_thread_pool_monitoring: bool = False

    terminate_worker_after_num_tasks: int | None = None
    disable_log_capture: bool = False
    log_queue_size: int = 1000
    grpc_enable_fork_support: bool = False
    force_shutdown_on_shutdown_signal: bool = False
    tenacity: TenacityConfig = TenacityConfig()

    @model_validator(mode="after")
    def validate_token_and_tenant(self) -> "ClientConfig":
        if not self.token:
            raise ValueError("Token must be set")

        if not self.token.startswith("ey"):
            raise ValueError(
                f"Token must be a valid JWT. Hint: These are the first few characters of the token provided: {self.token[:5]}"
            )

        if not self.tenant_id:
            self.tenant_id = get_tenant_id_from_jwt(self.token)

        return self

    @model_validator(mode="after")
    def validate_addresses(self) -> "ClientConfig":
        ## If nothing is set, read from the token
        ## If either is set, override what's in the JWT
        server_url_from_jwt, grpc_broadcast_address_from_jwt = get_addresses_from_jwt(
            self.token
        )

        if "host_port" not in self.model_fields_set:
            self.host_port = grpc_broadcast_address_from_jwt

        if "server_url" not in self.model_fields_set:
            self.server_url = server_url_from_jwt

        if not self.tls_config.server_name:
            self.tls_config.server_name = self.host_port.split(":")[0]

        if not self.tls_config.server_name:
            self.tls_config.server_name = "localhost"

        return self

    @field_validator("listener_v2_timeout")
    @classmethod
    def validate_listener_timeout(cls, value: int | None | str) -> int | None:
        if value is None:
            return None

        if isinstance(value, int):
            return value

        return int(value)

    @field_validator("namespace")
    @classmethod
    def validate_namespace(cls, namespace: str) -> str:
        if not namespace:
            return ""

        if not namespace.endswith("_"):
            namespace = f"{namespace}_"

        return namespace.lower()

    def __hash__(self) -> int:
        return hash(json.dumps(self.model_dump(), default=str))

    @overload
    def apply_namespace(
        self, resource_name: str, namespace_override: str | None = None
    ) -> str: ...

    @overload
    def apply_namespace(
        self, resource_name: None, namespace_override: str | None = None
    ) -> None: ...

    def apply_namespace(
        self, resource_name: str | None, namespace_override: str | None = None
    ) -> str | None:
        if resource_name is None:
            return None

        namespace = namespace_override or self.namespace

        if not namespace:
            return resource_name

        if resource_name.startswith(namespace):
            return resource_name

        return namespace + resource_name
