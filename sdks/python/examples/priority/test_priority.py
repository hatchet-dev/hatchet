import asyncio
from datetime import datetime, timedelta
from random import choice
from typing import Literal
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.priority.worker import DEFAULT_PRIORITY, priority_workflow
from hatchet_sdk import Hatchet, ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions
from hatchet_sdk.workflow_run import WorkflowRunRef

Priority = Literal["low", "medium", "high", "default"]


class RunPriorityStartedAt(BaseModel):
    priority: Priority
    started_at: datetime
    finished_at: datetime


def priority_to_int(priority: Priority) -> int:
    match priority:
        case "high":
            return 3
        case "medium":
            return 2
        case "low":
            return 1
        case "default":
            return DEFAULT_PRIORITY
        case _:
            raise ValueError(f"Invalid priority: {priority}")


@pytest.mark.asyncio()
async def test_priority(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    choices: list[Priority] = ["low", "medium", "high", "default"]

    run_refs = await priority_workflow.aio_run_many_no_wait(
        [
            priority_workflow.create_bulk_run_item(
                options=TriggerWorkflowOptions(
                    priority=(priority_to_int(priority := choice(choices))),
                    additional_metadata={
                        "priority": priority,
                        "key": ix,
                        "test_run_id": test_run_id,
                    },
                )
            )
            for ix in range(30)
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

    for i in range(len(runs_ids_started_ats) - 1):
        curr = runs_ids_started_ats[i]
        nxt = runs_ids_started_ats[i + 1]

        """Run start times should be in order of priority"""
        assert priority_to_int(curr.priority) >= priority_to_int(nxt.priority)

        """Runs should proceed one at a time"""
        assert curr.finished_at <= nxt.finished_at
        assert nxt.finished_at >= nxt.started_at

        """Runs should finish after starting (this is mostly a test for engine datetime handling bugs)"""
        assert curr.finished_at >= curr.started_at


@pytest.mark.skip(reason="Skipping until priority is implemented for scheduling")
@pytest.mark.asyncio()
async def test_priority_via_scheduling(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    choices: list[Priority] = ["low", "medium", "high", "default"]
    run_at = datetime.now() + timedelta(seconds=3)

    versions = await asyncio.gather(
        *[
            priority_workflow.aio_schedule(
                run_at=run_at,
                options=ScheduleTriggerWorkflowOptions(
                    priority=(priority_to_int(priority := choice(choices))),
                    additional_metadata={
                        "priority": priority,
                        "key": ix,
                        "test_run_id": test_run_id,
                    },
                ),
            )
            for ix in range(30)
        ]
    )

    await asyncio.sleep(15)

    runs = await hatchet.runs.aio_list(
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

    assert len(runs_ids_started_ats) == len(versions)

    for i in range(len(runs_ids_started_ats) - 1):
        curr = runs_ids_started_ats[i]
        nxt = runs_ids_started_ats[i + 1]

        """Run start times should be in order of priority"""
        assert priority_to_int(curr.priority) >= priority_to_int(nxt.priority)

        """Runs should proceed one at a time"""
        assert curr.finished_at <= nxt.finished_at
        assert nxt.finished_at >= nxt.started_at

        """Runs should finish after starting (this is mostly a test for engine datetime handling bugs)"""
        assert curr.finished_at >= curr.started_at
