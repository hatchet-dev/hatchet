import asyncio
import signal
from subprocess import Popen
from typing import Any

import psutil
import pytest

from examples.bug_tests.worker_pause_on_sigterm.worker import long_sleep, WORKER_NAME
from hatchet_sdk import EmptyModel, Hatchet, RunStatus


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
    ref = await long_sleep.aio_run(input=EmptyModel(), wait_for_result=False)

    for _ in range(30):
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)
        if run.status == RunStatus.RUNNING:
            break
        await asyncio.sleep(1)
    else:
        assert False, "Task never started running"

    parent = psutil.Process(on_demand_worker.pid)
    parent.send_signal(signal.SIGTERM)

    matching = []
    for _ in range(30):
        worker_list = await hatchet.workers.aio_list()

        matching = [
            w
            for w in (worker_list.rows or [])
            if w.name == WORKER_NAME and w.status == "PAUSED"
        ]

        if matching:
            break
        await asyncio.sleep(1)
    else:
        assert False, f"Worker {WORKER_NAME} never reported PAUSED"

    for _ in range(30):
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)
        if run.status == RunStatus.COMPLETED:
            break
        await asyncio.sleep(1)
    else:
        assert False, "Task never completed"
