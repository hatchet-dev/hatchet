import asyncio
import os
import signal
from pathlib import Path

import psutil
import pytest

from examples.bug_tests.worker_shutdown_no_premature_reassignment.worker import (
    SLEEP_SECONDS,
    drain_task,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import Hatchet, RunStatus, V1TaskStatus
from tests.worker_fixture import get_free_port, hatchet_worker

WORKER_A_NAME = "shutdown-drain-worker-a"
WORKER_B_NAME = "shutdown-drain-worker-b"

COMMAND = [
    "poetry",
    "run",
    "python",
    "examples/bug_tests/worker_shutdown_no_premature_reassignment/worker.py",
]


@pytest.mark.asyncio(loop_scope="session")
async def test_in_flight_task_completes_on_original_worker_without_reassignment(
    hatchet: Hatchet, tmp_path: Path
) -> None:
    """
    A worker that receives SIGTERM must keep heartbeating until its in-flight
    task has actually finished draining. If it stops heartbeating as soon as it
    stops reading new actions, the engine considers it dead ~30s later and
    reassigns the still-running task to another worker, causing the task to run
    a second time even though the first run was about to succeed.
    """
    log_path = tmp_path / "executions.log"
    log_path.write_text("")

    os.environ["SHUTDOWN_TEST_LOG_PATH"] = str(log_path)
    os.environ["HATCHET_TEST_WORKER_NAME"] = WORKER_A_NAME

    with hatchet_worker(COMMAND, get_free_port()) as worker_a_proc:
        ref = await drain_task.aio_run(wait_for_result=False)
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)

        await wait_for_running_status(hatchet, ref.workflow_run_id, timeout=30.0)

        # Worker B stays idle for the rest of the test, available to (wrongly)
        # steal the task if the engine ever decides worker A is dead.
        os.environ["HATCHET_TEST_WORKER_NAME"] = WORKER_B_NAME

        with hatchet_worker(COMMAND, get_free_port()):
            psutil.Process(worker_a_proc.pid).send_signal(signal.SIGTERM)

            for _ in range(30):
                worker_list = await hatchet.workers.aio_list()
                paused = [
                    w
                    for w in (worker_list or [])
                    if w.name == hatchet.config.apply_namespace(WORKER_A_NAME)
                    and w.status == "PAUSED"
                ]
                if paused:
                    break
                await asyncio.sleep(1)
            else:
                assert False, f"Worker {WORKER_A_NAME} never reported PAUSED"

            for _ in range(SLEEP_SECONDS + 60):
                run = await hatchet.runs.aio_get_details(ref.workflow_run_id)
                if run.status == RunStatus.COMPLETED:
                    break
                await asyncio.sleep(1)
            else:
                assert False, f"Task never completed, status was {run.status}"

    lines = log_path.read_text().splitlines()
    assert lines == [f"{WORKER_A_NAME} START", f"{WORKER_A_NAME} FINISH"], (
        "task must run exactly once, to completion, on the worker it started on "
        f"({WORKER_A_NAME}), even though that worker began shutting down while "
        f"the task was still in flight; got: {lines}"
    )

    completed_run = await hatchet.runs.aio_get(ref.workflow_run_id)
    assert len(completed_run.tasks) == 1

    task = completed_run.tasks[0]
    assert task.status == V1TaskStatus.COMPLETED
    assert task.attempt == 1, "task must not have been retried/reassigned"
    assert task.retry_count == 0, "task must not have been retried/reassigned"
