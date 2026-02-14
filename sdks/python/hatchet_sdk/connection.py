import os
from typing import Literal, cast, overload, Callable, TypeVar

import grpc

from hatchet_sdk.config import ClientConfig
from hatchet_sdk.exceptions import HatchetError


T = TypeVar("T")


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
        ("grpc.default_compression_algorithm", grpc.Compression.Gzip),
    ]

    os.environ["GRPC_ENABLE_FORK_SUPPORT"] = str(config.grpc_enable_fork_support)

    if config.grpc_enable_fork_support:
        os.environ["GRPC_POLL_STRATEGY"] = "poll"

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


# -------------------------------------------------------------------
# gRPC execution + transport error normalization (no behavior change)
# -------------------------------------------------------------------


def _execute_grpc_call(
    fn: Callable[..., T],
    *args,
    **kwargs,
) -> T:
    """
    Executes a gRPC stub call and normalizes transport-level errors.

    Does NOT change retry behavior.
    Only translates grpc.RpcError into clearer SDK-level exceptions.
    """
    try:
        return fn(*args, **kwargs)
    except grpc.RpcError as e:
        raise _translate_grpc_error(e) from e


def _translate_grpc_error(e: grpc.RpcError) -> Exception:
    """
    Translate grpc.RpcError into more specific SDK exceptions.

    No retry behavior changes.
    """
    code = e.code()

    # Transport-level / connectivity issues
    if code == grpc.StatusCode.DEADLINE_EXCEEDED:
        return GRPCTimeoutError(str(e))

    if code == grpc.StatusCode.UNAVAILABLE:
        return GRPCUnavailableError(str(e))

    # Fallback: preserve original behavior
    return e


# -------------------------------------------------------------------
# Transport exception types (minimal, incremental)
# -------------------------------------------------------------------


class GRPCTransportError(HatchetError):
    """Base class for gRPC transport-level failures."""


class GRPCTimeoutError(GRPCTransportError):
    """Raised when a gRPC call exceeds its deadline."""


class GRPCUnavailableError(GRPCTransportError):
    """Raised when the gRPC server is unavailable."""
