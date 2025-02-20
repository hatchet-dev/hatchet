from hatchet_sdk.loader import ClientConfig


def test_client_initialization_from_defaults() -> None:
    assert isinstance(ClientConfig(), ClientConfig)


def test_client_host_port_overrides() -> None:
    host_port = "localhost:8080"
    with_host_port = ClientConfig(host_port=host_port)
    assert with_host_port.host_port == host_port
    assert with_host_port.server_url == host_port

    assert ClientConfig().host_port != host_port
    assert ClientConfig().server_url != host_port
