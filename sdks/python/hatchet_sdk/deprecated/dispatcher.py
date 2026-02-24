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
from hatchet_sdk.deprecated.action_listener import LegacyGetActionListenerRequest
from hatchet_sdk.metadata import get_metadata

DEFAULT_REGISTER_TIMEOUT = 30


async def legacy_get_action_listener(
    config: ClientConfig,
    req: LegacyGetActionListenerRequest,
) -> ActionListener:
    """Register a worker using the legacy slots field (for pre-slot-config engines)."""
    aio_conn = new_conn(config, True)
    aio_client = DispatcherStub(aio_conn)

    # Override labels with the preset labels
    preset_labels = config.worker_preset_labels

    for key, value in preset_labels.items():
        req.labels[key] = WorkerLabels(str_value=str(value))

    response = cast(
        WorkerRegisterResponse,
        await aio_client.Register(  # type: ignore[misc]
            WorkerRegisterRequest(
                worker_name=req.worker_name,
                actions=req.actions,
                services=req.services,
                slots=req.slots,
                labels=req.labels,
                runtime_info=RuntimeInfo(
                    sdk_version=version("hatchet_sdk"),
                    language=SDKS.PYTHON,
                    language_version=f"{version_info.major}.{version_info.minor}.{version_info.micro}",
                    os=platform.system().lower(),
                ),
            ),
            timeout=DEFAULT_REGISTER_TIMEOUT,
            metadata=get_metadata(config.token),
        ),
    )

    return ActionListener(config, response.worker_id)
