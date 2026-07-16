from datetime import datetime
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest

from tests.slot_cost.workflow import WORKER_SLOTS, slot_cost_test_heavy_task


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py", "--slots", str(WORKER_SLOTS)]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_slot_cost_blocks_second_run_until_first_finishes(
    on_demand_worker: Popen[Any],
) -> None:
    test_run_id = str(uuid4())

    await slot_cost_test_heavy_task.aio_run_many(
        [
            slot_cost_test_heavy_task.create_bulk_run_item(
                additional_metadata={"test_run_id": test_run_id},
            )
            for _ in range(2)
        ]
    )

    runs = await slot_cost_test_heavy_task.aio_list_runs(
        additional_metadata={"test_run_id": test_run_id}
    )

    assert len(runs) == 2

    first, second = sorted(runs, key=lambda r: r.started_at or datetime.max)

    assert first.started_at is not None
    assert first.finished_at is not None
    assert second.started_at is not None

    # The worker has 2 * SLOT_COST - 1 slots, so while the first run holds SLOT_COST of them the
    # remaining SLOT_COST - 1 cannot fit the second run.
    assert second.started_at >= first.finished_at
