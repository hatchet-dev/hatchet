import asyncio
from datetime import datetime
from random import choice
from typing import Literal
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.priority.worker import priority_workflow
from hatchet_sdk import Hatchet, TriggerWorkflowOptions


class RunPriorityStartedAt(BaseModel):
    priority: Literal["low", "high"]
    started_at: datetime
    finished_at: datetime


@pytest.mark.asyncio()
async def test_priority(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    run_refs = await priority_workflow.aio_run_many_no_wait(
        [
            priority_workflow.create_bulk_run_item(
                options=TriggerWorkflowOptions(
                    priority=(
                        3 if (priority := choice(["low", "high"])) == "high" else 1
                    ),
                    additional_metadata={
                        "priority": priority,
                        "key": ix,
                        "test_run_id": test_run_id,
                    },
                )
            )
            for ix in range(20)
        ]
    )

    await asyncio.gather(*[r.aio_result() for r in run_refs])

    workflows = (
        await hatchet.workflows.aio_list(workflow_name=priority_workflow.name)
    ).rows

    assert workflows

    workflow = next((w for w in workflows if w.name == priority_workflow.name), None)

    assert workflow

    assert workflow.name == priority_workflow.name

    runs = await hatchet.runs.aio_list(
        workflow_ids=[workflow.metadata.id],
        additional_metadata={
            "test_run_id": test_run_id,
        },
    )

    runs_ids_started_ats: list[RunPriorityStartedAt] = sorted(
        [
            RunPriorityStartedAt(
                priority=(r.additional_metadata or {}).get("priority") or "low",
                started_at=r.started_at or datetime.min,
                finished_at=r.finished_at or datetime.min,
            )
            for r in runs.rows
        ],
        key=lambda x: x.started_at,
    )

    assert len(runs_ids_started_ats) == len(run_refs)

    has_seen_low_priority_run = None

    for run in runs_ids_started_ats:
        if run.priority == "low" and has_seen_low_priority_run is None:
            has_seen_low_priority_run = True

        if has_seen_low_priority_run is True:
            assert run.priority == "low"
        else:
            assert run.priority == "high"

    low_prio_runs = [r for r in runs_ids_started_ats if r.priority == "low"]
    high_prio_runs = [r for r in runs_ids_started_ats if r.priority == "high"]

    assert max(r.finished_at for r in high_prio_runs) < min(
        r.started_at for r in low_prio_runs
    )
