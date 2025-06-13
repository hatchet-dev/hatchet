import os
from typing import Literal, cast, overload

import grpc

from hatchet_sdk.config import ClientConfig


@overload
def new_conn(config: ClientConfig, aio: Literal[False]) -> grpc.Channel: ...


@overload
def new_conn(config: ClientConfig, aio: Literal[True]) -> grpc.aio.Channel: ...


def new_conn(config: ClientConfig, aio: bool) -> grpc.Channel | grpc.aio.Channel:
    credentials: grpc.ChannelCredentials | None = None

    # load channel credentials
    if config.tls_config.strategy == "tls":
        root: bytes | None = None

        if config.tls_config.root_ca_file:
            with open(config.tls_config.root_ca_file, "rb") as f:
                root = f.read()

        credentials = grpc.ssl_channel_credentials(root_certificates=root)
    elif config.tls_config.strategy == "mtls":
        assert config.tls_config.root_ca_file
        assert config.tls_config.key_file
        assert config.tls_config.cert_file

        with open(config.tls_config.root_ca_file, "rb") as f:
            root = f.read()

        with open(config.tls_config.key_file, "rb") as f:
            private_key = f.read()

        with open(config.tls_config.cert_file, "rb") as f:
            certificate_chain = f.read()

        credentials = grpc.ssl_channel_credentials(
            root_certificates=root,
            private_key=private_key,
            certificate_chain=certificate_chain,
        )

    start = grpc if not aio else grpc.aio

    channel_options: list[tuple[str, str | int]] = [
        ("grpc.max_send_message_length", config.grpc_max_send_message_length),
        ("grpc.max_receive_message_length", config.grpc_max_recv_message_length),
        ("grpc.keepalive_time_ms", 10 * 1000),
        ("grpc.keepalive_timeout_ms", 60 * 1000),
        ("grpc.client_idle_timeout_ms", 60 * 1000),
        ("grpc.http2.max_pings_without_data", 0),
        ("grpc.keepalive_permit_without_calls", 1),
    ]

    # Set environment variable to disable fork support. Reference: https://github.com/grpc/grpc/issues/28557
    # When steps execute via os.fork, we see `TSI_DATA_CORRUPTED` errors.
    os.environ["GRPC_ENABLE_FORK_SUPPORT"] = "False"

    if config.tls_config.strategy == "none":
        conn = start.insecure_channel(
            target=config.host_port,
            options=channel_options,
        )
    else:
        channel_options.append(
            ("grpc.ssl_target_name_override", config.tls_config.server_name)
        )

        conn = start.secure_channel(
            target=config.host_port,
            credentials=credentials,
            options=channel_options,
        )

    return cast(
        grpc.Channel | grpc.aio.Channel,
        conn,
    )
