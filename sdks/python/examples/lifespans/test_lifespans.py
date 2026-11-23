import asyncio
import contextlib
import signal
from subprocess import Popen
from typing import Any

import psutil
import pytest

from examples.lifespans.drain import DrainInput, lifespan_drain_task
from examples.lifespans.simple import Lifespan, lifespan_task
from examples.test_utils import wait_for_run_status
from hatchet_sdk import Hatchet


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

    await wait_for_run_status(hatchet, ref.workflow_run_id, "RUNNING", timeout=30)

    await asyncio.sleep(1)

    parent = psutil.Process(on_demand_worker.pid)
    children = parent.children(recursive=True)

    for proc in [parent, *children]:
        with contextlib.suppress(psutil.NoSuchProcess):
            proc.send_signal(signal.SIGTERM)

    result = await ref.aio_result()

    assert result.status == "ok"
    assert result.iterations == n
