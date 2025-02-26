import os
from typing import TYPE_CHECKING

import grpc

if TYPE_CHECKING:
    from hatchet_sdk.v0.loader import ClientConfig


def new_conn(config: "ClientConfig", aio=False):
    credentials: grpc.ChannelCredentials | None = None

    # load channel credentials
    if config.tls_config.tls_strategy == "tls":
        root: bytes | None = None

        if config.tls_config.ca_file:
            root = open(config.tls_config.ca_file, "rb").read()

        credentials = grpc.ssl_channel_credentials(root_certificates=root)
    elif config.tls_config.tls_strategy == "mtls":
        root = open(config.tls_config.ca_file, "rb").read()
        private_key = open(config.tls_config.key_file, "rb").read()
        certificate_chain = open(config.tls_config.cert_file, "rb").read()

        credentials = grpc.ssl_channel_credentials(
            root_certificates=root,
            private_key=private_key,
            certificate_chain=certificate_chain,
        )

    start = grpc if not aio else grpc.aio

    channel_options = [
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

    if config.tls_config.tls_strategy == "none":
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
    return conn
