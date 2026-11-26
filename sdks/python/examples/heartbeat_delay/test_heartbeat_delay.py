from datetime import timedelta
from subprocess import Popen
from typing import Any

import grpc
import pytest

from examples.heartbeat_delay.worker import (
    MAX_HEARTBEAT_DELAY_BEFORE_DECLARED_DEAD,
    WORKER_NAME,
)
from examples.test_utils import poll_until
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.dispatcher.dispatcher import DispatcherClient
from hatchet_sdk.clients.rest.models.worker import Worker
from hatchet_sdk.clients.rest.models.worker_status import WorkerStatus


def find_active_worker(workers: list[Worker], worker_name: str) -> Worker | None:
    return next(
        (
            w
            for w in workers
            if w.name == worker_name and w.status == WorkerStatus.ACTIVE
        ),
        None,
    )


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "examples/heartbeat_delay/worker.py"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_worker_reports_configured_max_heartbeat_delay(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    namespaced_worker_name = hatchet.config.apply_namespace(WORKER_NAME)

    workers = await poll_until(
        hatchet.workers.aio_list,
        lambda workers: find_active_worker(workers, namespaced_worker_name) is not None,
        description=f"worker {WORKER_NAME} to be active",
        timeout=30,
    )
    worker = find_active_worker(workers, namespaced_worker_name)

    assert worker is not None
    assert worker.max_heartbeat_delay_before_declared_dead_seconds == int(
        MAX_HEARTBEAT_DELAY_BEFORE_DECLARED_DEAD.total_seconds()
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_registration_rejects_max_heartbeat_delay_below_heartbeat_interval(
    hatchet: Hatchet,
) -> None:
    dispatcher = DispatcherClient(hatchet.config)

    with pytest.raises(grpc.aio.AioRpcError) as error:
        await dispatcher.get_action_listener(
            worker_name="heartbeat-delay-too-short-worker",
            services=["default"],
            actions=[],
            slot_config={"default": 1},
            labels=[],
            max_heartbeat_delay_before_declared_dead=timedelta(seconds=2),
        )

    assert error.value.code() == grpc.StatusCode.INVALID_ARGUMENT
