from subprocess import Popen
from typing import Any

import pytest
import tenacity
from tenacity.wait import wait_fixed

from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.worker_status import WorkerStatus
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup
from tests.zombie_worker.worker import die
import uuid

_worker_name = f"e2e-test-zombie-worker-{uuid.uuid4()}"


@pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "tests/zombie_worker/worker.py",
            "--name",
            _worker_name,
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_zombie_worker(hatchet: Hatchet, on_demand_worker: Popen[Any]) -> None:
    workers = await hatchet.workers.aio_list()
    assert workers.rows
    worker = next((w for w in workers.rows if w.name == _worker_name), None)
    assert worker is not None
    worker_id = worker.metadata.id
    with pytest.raises(FailedTaskRunExceptionGroup):
        await die.aio_run(desired_worker_id=worker.metadata.id)

    @tenacity.retry(stop=tenacity.stop_after_delay(5), wait=wait_fixed(0.5))
    async def wait_for_worker_inactive() -> None:
        worker = await hatchet.workers.aio_get(worker_id)
        assert worker.status == WorkerStatus.INACTIVE

    await wait_for_worker_inactive()
