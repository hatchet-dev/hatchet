import signal
from subprocess import Popen
from typing import Any

import psutil
import pytest

from examples.bug_tests.worker_pause_on_sigterm.worker import WORKER_NAME, long_sleep
from examples.test_utils import wait_for_run_status, wait_for_worker_status
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.worker_status import WorkerStatus


@pytest.mark.parametrize(
    "on_demand_worker",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/bug_tests/worker_pause_on_sigterm/worker.py",
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_worker_pauses_when_only_parent_receives_sigterm(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    ref = await long_sleep.aio_run(wait_for_result=False)

    await wait_for_run_status(hatchet, ref.workflow_run_id, "RUNNING", timeout=30)

    parent = psutil.Process(on_demand_worker.pid)
    parent.send_signal(signal.SIGTERM)

    await wait_for_worker_status(hatchet, WORKER_NAME, WorkerStatus.PAUSED)

    await wait_for_run_status(hatchet, ref.workflow_run_id, "COMPLETED", timeout=30)
