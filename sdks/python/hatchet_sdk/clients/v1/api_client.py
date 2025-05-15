from typing import ParamSpec, TypeVar, overload

from hatchet_sdk.clients.rest.api_client import ApiClient
from hatchet_sdk.clients.rest.configuration import Configuration
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.utils.namespacing import apply_namespace
from hatchet_sdk.utils.typing import JSONSerializableMapping

## Type variables to use with coroutines.
## See https://stackoverflow.com/questions/73240620/the-right-way-to-type-hint-a-coroutine-function
## Return type
R = TypeVar("R")

## Yield type
Y = TypeVar("Y")

## Send type
S = TypeVar("S")

P = ParamSpec("P")


def maybe_additional_metadata_to_kv(
    additional_metadata: dict[str, str] | JSONSerializableMapping | None
) -> list[str] | None:
    if not additional_metadata:
        return None

    return [f"{k}:{v}" for k, v in additional_metadata.items()]


class BaseRestClient:
    def __init__(self, config: ClientConfig) -> None:
        self.tenant_id = config.tenant_id

        self.client_config = config
        self.api_config = Configuration(
            host=config.server_url,
            access_token=config.token,
        )

        self.api_config.datetime_format = "%Y-%m-%dT%H:%M:%S.%fZ"

    def client(self) -> ApiClient:
        return ApiClient(self.api_config)

    @overload
    def apply_namespace(self, resource_name: str) -> str: ...

    @overload
    def apply_namespace(self, resource_name: None) -> None: ...

    def apply_namespace(self, resource_name: str | None) -> str | None:
        return apply_namespace(
            resource_name=resource_name,
            namespace=self.client_config.namespace,
        )
