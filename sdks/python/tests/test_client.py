import os
from unittest import mock

from hatchet_sdk.config import ClientConfig


def test_client_initialization_from_defaults() -> None:
    assert isinstance(ClientConfig(), ClientConfig)


def test_client_host_port_overrides() -> None:
    host_port = "foo:8080"
    with_host_port = ClientConfig(host_port=host_port)

    assert with_host_port.host_port == host_port
    assert with_host_port.server_url == ClientConfig().server_url

    assert ClientConfig().host_port != host_port
    assert ClientConfig().server_url != host_port


def test_client_host_port_override_when_env_var() -> None:
    with mock.patch.dict(os.environ, {"HATCHET_CLIENT_HOST_PORT": "foo:8080"}):
        config = ClientConfig()

    assert config.host_port == "foo:8080"
    assert config.server_url == ClientConfig().server_url


def test_client_server_url_override_when_env_var() -> None:
    with mock.patch.dict(os.environ, {"HATCHET_CLIENT_SERVER_URL": "foobaz:8080"}):
        config = ClientConfig()

    assert config.server_url == "foobaz:8080"
    assert config.host_port == ClientConfig().host_port
