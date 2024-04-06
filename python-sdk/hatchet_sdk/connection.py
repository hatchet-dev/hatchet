from typing import Any

import grpc


def new_conn(config, aio=False):

    credentials: grpc.ChannelCredentials | None = None

    # load channel credentials
    if config.tls_config.tls_strategy == "tls":
        root: Any | None = None

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

    strat = grpc if not aio else grpc.aio

    if config.tls_config.tls_strategy == "none":
        conn = strat.insecure_channel(
            target=config.host_port,
        )
    else:
        conn = strat.secure_channel(
            target=config.host_port,
            credentials=credentials,
            options=[("grpc.ssl_target_name_override", config.tls_config.server_name)],
        )
    return conn
