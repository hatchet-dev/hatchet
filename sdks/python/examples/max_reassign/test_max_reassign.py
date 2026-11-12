import os

import psutil
import pytest

from examples.max_reassign.worker import (
    FIRST_WORKER_NAME,
    non_reassignable_task,
    reassignable_task,
)
from examples.test_utils import wait_for_running_status
from hatchet_sdk import FailedTaskRunExceptionGroup, Hatchet
from tests.worker_fixture import get_free_port, hatchet_worker

SECOND_WORKER_NAME = "max-reassign-worker-b"

COMMAND = ["poetry", "run", "python", "examples/max_reassign/worker.py"]


def kill_process_tree(pid: int) -> None:
    process = psutil.Process(pid)

    for child in process.children(recursive=True):
        child.kill()

    process.kill()


@pytest.mark.asyncio(loop_scope="session")
async def test_task_fails_instead_of_being_reassigned_after_max_reassignments(
    hatchet: Hatchet,
) -> None:
    os.environ["HATCHET_TEST_WORKER_NAME"] = FIRST_WORKER_NAME

    with hatchet_worker(COMMAND, get_free_port()) as first_worker:
        reassignable_ref = await reassignable_task.aio_run(wait_for_result=False)
        non_reassignable_ref = await non_reassignable_task.aio_run(
            wait_for_result=False
        )

        await wait_for_running_status(hatchet, reassignable_ref.workflow_run_id)
        await wait_for_running_status(hatchet, non_reassignable_ref.workflow_run_id)

        os.environ["HATCHET_TEST_WORKER_NAME"] = SECOND_WORKER_NAME

        with hatchet_worker(COMMAND, get_free_port()):
            kill_process_tree(first_worker.pid)

            reassigned_result = await reassignable_ref.aio_result()
            assert reassigned_result.worker_name == SECOND_WORKER_NAME

            with pytest.raises(
                FailedTaskRunExceptionGroup,
                match="Task reached its maximum reassignment count",
            ):
                await non_reassignable_ref.aio_result()
