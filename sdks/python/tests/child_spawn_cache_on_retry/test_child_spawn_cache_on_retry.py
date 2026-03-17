from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest

from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from tests.child_spawn_cache_on_retry.worker import (
    spawn_cache_on_retry_child,
    spawn_cache_on_retry_parent,
)


@pytest.mark.parametrize(
    "on_demand_worker",
    [(["poetry", "run", "python", "tests/worker.py", "--slots", "5"], 8005)],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_spawn_caching_on_retry(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    test_run_id = str(uuid4())
    try:
        await spawn_cache_on_retry_parent.aio_run(
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id}
            )
        )
    except Exception as e:
        assert "Task exceeded timeout of" in str(e)

    runs = await spawn_cache_on_retry_child.aio_list_runs(
        additional_metadata={"test_run_id": test_run_id}
    )

    assert len(runs) == 1

    run = runs[0]

    assert run.status == V1TaskStatus.COMPLETED
