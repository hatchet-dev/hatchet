"""Legacy dispatcher registration using slots: int (pre-slot-config engines)."""

import platform
from importlib.metadata import version
from sys import version_info
from typing import cast

from hatchet_sdk.clients.dispatcher.action_listener import ActionListener
from hatchet_sdk.config import ClientConfig
from hatchet_sdk.connection import new_conn
from hatchet_sdk.contracts.dispatcher_pb2 import (
    SDKS,
    RuntimeInfo,
    WorkerLabels,
    WorkerRegisterRequest,
    WorkerRegisterResponse,
)
from hatchet_sdk.contracts.dispatcher_pb2_grpc import DispatcherStub
from hatchet_sdk.types.labels import WorkerLabel
from hatchet_sdk.utils.api_auth import create_authorization_header

DEFAULT_REGISTER_TIMEOUT = 30


async def legacy_get_action_listener(
    config: ClientConfig,
    worker_name: str,
    services: list[str],
    actions: list[str],
    slots: int,
    labels: list[WorkerLabel],
) -> ActionListener:
    """Register a worker using the legacy slots field (for pre-slot-config engines)."""
    aio_conn = new_conn(config, True)
    aio_client = DispatcherStub(aio_conn)

    proto_labels: dict[str, WorkerLabels] = {
        label.key: label.to_proto() for label in labels if label.key is not None
    }
    for key, value in config.worker_preset_labels.items():
        proto_labels[key] = WorkerLabels(str_value=str(value))

    response = cast(
        WorkerRegisterResponse,
        await aio_client.Register(  # type: ignore[misc]
            WorkerRegisterRequest(
                worker_name=worker_name,
                actions=actions,
                services=services,
                slots=slots,
                labels=proto_labels,
                runtime_info=RuntimeInfo(
                    sdk_version=version("hatchet_sdk"),
                    language=SDKS.PYTHON,
                    language_version=f"{version_info.major}.{version_info.minor}.{version_info.micro}",
                    os=platform.system().lower(),
                ),
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=create_authorization_header(config.token),
        ),
    )

    return ActionListener(config, response.worker_id)
