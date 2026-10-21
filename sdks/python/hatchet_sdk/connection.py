import base64
import functools
import hashlib
import os
import pkgutil
import re
from typing import Literal, cast, overload

import grpc

from hatchet_sdk.config import ClientConfig

DISTRUSTED_ROOT_SHA256 = frozenset(
    {
        "86a1ecba089c4a8d3bbe2734c612ba341d813e043cf9e8a862cd5c57a36bbe6b",
        "d7a7a0fb5d7e2731d771e9484ebcdef71d5f0c3e0a2948782bc83ee0ea699ef4",
    }
)

_PEM_BLOCK = re.compile(
    rb"-----BEGIN CERTIFICATE-----.+?-----END CERTIFICATE-----", re.DOTALL
)


def _is_distrusted(pem_block: bytes) -> bool:
    body = b"".join(pem_block.splitlines()[1:-1])
    try:
        der = base64.b64decode(body)
    except ValueError:
        return False

    return hashlib.sha256(der).hexdigest() in DISTRUSTED_ROOT_SHA256


def filter_distrusted_roots(bundle: bytes) -> bytes:
    kept = [block for block in _PEM_BLOCK.findall(bundle) if not _is_distrusted(block)]

    return b"\n".join(kept) + b"\n"


@functools.lru_cache(maxsize=1)
def _filtered_bundled_roots() -> bytes | None:
    try:
        bundle = pkgutil.get_data("grpc", "_cython/_credentials/roots.pem")
    except Exception:
        return None

    if not bundle:
        return None

    filtered = filter_distrusted_roots(bundle)

    return filtered if _PEM_BLOCK.search(filtered) else None


def _uses_bundled_roots() -> bool:
    if os.environ.get("GRPC_DEFAULT_SSL_ROOTS_FILE_PATH"):
        return False

    system_roots = os.environ.get("GRPC_USE_SYSTEM_ROOTS_OVER_LANGUAGE_CALLBACK", "")

    return system_roots.strip().lower() not in {"1", "true", "yes", "on"}


def load_channel_credentials(config: ClientConfig) -> grpc.ChannelCredentials | None:
    credentials: grpc.ChannelCredentials | None = None

    # load channel credentials
    if config.tls_config.strategy == "tls":
        root: bytes | None = None

        if config.tls_config.root_ca_file:
            with open(config.tls_config.root_ca_file, "rb") as f:
                root = f.read()
        elif _uses_bundled_roots():
            root = _filtered_bundled_roots()

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

    return credentials


@overload
def new_conn(config: ClientConfig, aio: Literal[False]) -> grpc.Channel: ...


@overload
def new_conn(config: ClientConfig, aio: Literal[True]) -> grpc.aio.Channel: ...


def new_conn(config: ClientConfig, aio: bool) -> grpc.Channel | grpc.aio.Channel:
    credentials = load_channel_credentials(config)

    start = grpc if not aio else grpc.aio

    channel_options: list[tuple[str, str | int]] = [
        ("grpc.max_send_message_length", config.grpc_max_send_message_length),
        ("grpc.max_receive_message_length", config.grpc_max_recv_message_length),
        ("grpc.keepalive_time_ms", 10 * 1000),
        ("grpc.keepalive_timeout_ms", 60 * 1000),
        ("grpc.client_idle_timeout_ms", 60 * 1000),
        ("grpc.http2.max_pings_without_data", 0),
        ("grpc.keepalive_permit_without_calls", 1),
        ("grpc.default_compression_algorithm", grpc.Compression.Gzip),
    ]

    if not credentials:
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
