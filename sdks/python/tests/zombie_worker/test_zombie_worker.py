from subprocess import Popen
from typing import Any

import pytest

from hatchet_sdk import Hatchet
from tests.zombie_worker.worker import die


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py", "--slots", "1"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_zombie_worker(hatchet: Hatchet, on_demand_worker: Popen[Any]) -> None:
    await die.aio_run()
