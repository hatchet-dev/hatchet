from subprocess import Popen
from typing import Any

import pytest

from hatchet_sdk import Hatchet
from tests.child_spawn_cache_on_retry.worker import (
    spawn_cache_on_retry_child,
    spawn_cache_on_retry_parent,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py", "--slots", "5"]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_spawn_caching_on_retry(
    hatchet: Hatchet, on_demand_worker: Popen[Any], test_run_id: str
) -> None:
    with pytest.raises(Exception, match="Task exceeded timeout of"):
        await spawn_cache_on_retry_parent.aio_run(
            additional_metadata={"test_run_id": test_run_id}
        )

    runs = await spawn_cache_on_retry_child.aio_list_runs(
        additional_metadata={"test_run_id": test_run_id}
    )

    assert len(runs) == 1

    run = runs[0]

    assert run.status == "COMPLETED"
