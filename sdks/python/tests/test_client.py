import os
from unittest import mock

from hatchet_sdk.config import ClientConfig
from hatchet_sdk.utils.slots import resolve_worker_slot_config
from hatchet_sdk.worker.slot_types import SlotType


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


def test_resolve_slot_config_no_durable() -> None:
    resolved = resolve_worker_slot_config(
        slot_config=None,
        slots=None,
        durable_slots=None,
        workflows=None,
    )

    assert resolved == {SlotType.DEFAULT: 100}


def test_resolve_slot_config_only_durable() -> None:
    class DummyTask:
        is_durable = True
        slot_requests: dict[str, int] = {"durable": 1}

    class DummyWorkflow:
        tasks = [DummyTask()]

    resolved = resolve_worker_slot_config(
        slot_config=None,
        slots=None,
        durable_slots=None,
        workflows=[DummyWorkflow()],
    )

    assert resolved == {SlotType.DURABLE: 1000}


def test_resolve_slot_config_mixed() -> None:
    class DefaultTask:
        is_durable = False
        slot_requests: dict[str, int] = {"default": 1}

    class DurableTask:
        is_durable = True
        slot_requests: dict[str, int] = {"durable": 1}

    class DummyWorkflow:
        tasks = [DefaultTask(), DurableTask()]

    resolved = resolve_worker_slot_config(
        slot_config=None,
        slots=None,
        durable_slots=None,
        workflows=[DummyWorkflow()],
    )

    assert resolved == {SlotType.DEFAULT: 100, SlotType.DURABLE: 1000}
