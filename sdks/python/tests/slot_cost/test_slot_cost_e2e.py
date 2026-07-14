import asyncio
from datetime import datetime
from subprocess import Popen
from typing import Any
from uuid import uuid4

import pytest

from hatchet_sdk import EmptyModel, Hatchet
from tests.slot_cost.workflow import WORKER_SLOTS, slot_cost_workflow


@pytest.mark.parametrize(
    "on_demand_worker",
    [["poetry", "run", "python", "tests/worker.py", "--slots", str(WORKER_SLOTS)]],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_slot_cost_blocks_second_run_until_first_finishes(
    hatchet: Hatchet, on_demand_worker: Popen[Any]
) -> None:
    test_run_id = str(uuid4())

    refs = await slot_cost_workflow.aio_run_many(
        [
            slot_cost_workflow.create_bulk_run_item(
                EmptyModel(),
                additional_metadata={"test_run_id": test_run_id},
            )
            for _ in range(2)
        ],
        wait_for_result=False,
    )

    await asyncio.gather(*[r.aio_result() for r in refs])

    workflows = (
        await hatchet.workflows.aio_list(
            workflow_name=slot_cost_workflow.name,
            limit=100,
        )
    ).rows

    assert workflows

    workflow = next(w for w in workflows if w.name == slot_cost_workflow.name)

    runs = await hatchet.runs.aio_list(
        workflow_ids=[workflow.metadata.id],
        additional_metadata={"test_run_id": test_run_id},
        limit=100,
    )

    assert len(runs.rows) == 2

    first, second = sorted(runs.rows, key=lambda r: r.started_at or datetime.max)

    assert first.started_at is not None
    assert first.finished_at is not None
    assert second.started_at is not None

    # The worker has 2 * SLOT_COST - 1 slots, so while the first run holds SLOT_COST of them the
    # remaining SLOT_COST - 1 cannot fit the second run.
    assert second.started_at >= first.finished_at
