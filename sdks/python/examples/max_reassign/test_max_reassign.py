from subprocess import Popen
from typing import Any

import pytest

from examples.max_reassign.worker import (
    no_reassigns_allowed,
    WorkerIdEnvelope,
    worker_id_returner,
)
from hatchet_sdk import (
    Hatchet,
    DesiredWorkerLabel,
    FailedTaskRunExceptionGroup,
)
import pytest
from subprocess import Popen
from typing import Any, Generator
from conftest import _on_demand_worker_fixture


@pytest.fixture(scope="session")
def on_demand_worker_max_reassign_a(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.fixture(scope="session")
def on_demand_worker_max_reassign_b(
    request: pytest.FixtureRequest,
) -> Generator[Popen[bytes], None, None]:
    yield from _on_demand_worker_fixture(request)


@pytest.mark.parametrize(
    "on_demand_worker_max_reassign_a",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/max_reassign/worker.py",
            "--label",
            "a",
        ]
    ],
    indirect=True,
)
@pytest.mark.parametrize(
    "on_demand_worker_max_reassign_b",
    [
        [
            "poetry",
            "run",
            "python",
            "examples/max_reassign/worker.py",
            "--label",
            "b",
        ]
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_late_join_mid_stream_stays_open(
    hatchet: Hatchet,
    on_demand_worker_max_reassign_a: Popen[Any],
    on_demand_worker_max_reassign_b: Popen[Any],
) -> None:
    worker_a = await worker_id_returner.aio_run(
        desired_worker_labels=[
            DesiredWorkerLabel(key="no-reassign", value="a", required=True)
        ],
        wait_for_result=True,
    )

    worker_b = await worker_id_returner.aio_run(
        desired_worker_labels=[
            DesiredWorkerLabel(key="no-reassign", value="b", required=True)
        ],
        wait_for_result=True,
    )

    worker_id_to_pid = {
        worker_a.worker_id: on_demand_worker_max_reassign_a.pid,
        worker_b.worker_id: on_demand_worker_max_reassign_b.pid,
    }

    ref = await no_reassigns_allowed.aio_run(wait_for_result=False)

    data: WorkerIdEnvelope | None = None

    async for e in hatchet.runs.subscribe_to_stream(ref.workflow_run_id):
        data = WorkerIdEnvelope.model_validate_json(e)
        break

    assert data is not None, "worker id was not streamed back from the worker"

    pid = worker_id_to_pid[data.worker_id]

    if pid == on_demand_worker_max_reassign_a.pid:
        on_demand_worker_max_reassign_a.kill()

    if pid == on_demand_worker_max_reassign_b.pid:
        on_demand_worker_max_reassign_b.kill()

    with pytest.raises(
        FailedTaskRunExceptionGroup, match="Task exceeded timeout of 1m"
    ):
        await ref.aio_result()
