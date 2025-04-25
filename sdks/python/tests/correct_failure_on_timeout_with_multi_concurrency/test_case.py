# Put these switch cases after scheduling timed out case in all six places
# Write an e2e test that has two tasks with small scheduling timeouts,
#     and a worker with no slots available, and have the tasks have the same concurrency strategy
# And make sure that the tasks are marked as failed correctly over the API once the scheduling timeout is reached

from subprocess import Popen

import pytest

from hatchet_sdk import Hatchet


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_failure_on_timeout(hatchet: Hatchet, on_demand_worker: Popen) -> None:
    assert True
