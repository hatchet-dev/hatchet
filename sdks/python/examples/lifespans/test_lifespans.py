import asyncio
import signal
from subprocess import Popen
from typing import Any

import psutil
import pytest

from examples.lifespans.drain import lifespan_drain_task, DrainInput
from examples.lifespans.simple import Lifespan, lifespan_task
from hatchet_sdk import Hatchet
from hatchet_sdk import RunStatus


@pytest.mark.asyncio(loop_scope="session")
async def test_lifespans() -> None:
    result = await lifespan_task.aio_run()

    assert isinstance(result, Lifespan)
    assert result.pi == 3.14
    assert result.foo == "bar"


@pytest.mark.parametrize(
    "on_demand_worker",
    [
        ["poetry", "run", "python", "examples/lifespans/drain.py"],
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_lifespan_drain_on_sigterm(
    hatchet: Hatchet,
    on_demand_worker: Popen[Any],
) -> None:
    n = 6
    ref = await lifespan_drain_task.aio_run(DrainInput(n=n), wait_for_result=False)

    for _ in range(30):
        run = await hatchet.runs.aio_get_details(ref.workflow_run_id)
        if run.status == RunStatus.RUNNING:
            break

        await asyncio.sleep(1)
    else:
        pytest.fail("Task never entered RUNNING state")

    await asyncio.sleep(1)

    parent = psutil.Process(on_demand_worker.pid)
    children = parent.children(recursive=True)

    for proc in [parent] + children:
        try:
            proc.send_signal(signal.SIGTERM)
        except psutil.NoSuchProcess:
            pass

    result = await ref.aio_result()

    assert result.status == "ok"
    assert result.iterations == n
