import json
from logging import Logger, getLogger

from pydantic import Field, field_validator, model_validator
from pydantic_settings import BaseSettings, SettingsConfigDict

from hatchet_sdk.token import get_addresses_from_jwt, get_tenant_id_from_jwt


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

    listener_v2_timeout: int | None = None
    grpc_max_recv_message_length: int = Field(
        default=4 * 1024 * 1024, description="4MB default"
    )
    grpc_max_send_message_length: int = Field(
        default=4 * 1024 * 1024, description="4MB default"
    )

    worker_preset_labels: dict[str, str] = Field(default_factory=dict)
    enable_force_kill_sync_threads: bool = False

    @model_validator(mode="after")
    def validate_token_and_tenant(self) -> "ClientConfig":
        if not self.token:
            raise ValueError("Token must be set")

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
