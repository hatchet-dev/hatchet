import asyncio
from collections.abc import AsyncGenerator
from datetime import datetime, timedelta, timezone
from random import choice
from subprocess import Popen
from typing import Any, Literal
from uuid import uuid4

import pytest
import pytest_asyncio
from pydantic import BaseModel

from examples.priority.worker import DEFAULT_PRIORITY, SLEEP_TIME, priority_workflow
from hatchet_sdk import Hatchet, ScheduleTriggerWorkflowOptions, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

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


@pytest_asyncio.fixture(loop_scope="session", scope="function")
async def dummy_runs() -> None:
    priority: Priority = "high"

    await priority_workflow.aio_run_many_no_wait(
        [
            priority_workflow.create_bulk_run_item(
                options=TriggerWorkflowOptions(
                    priority=(priority_to_int(priority)),
                    additional_metadata={
                        "priority": priority,
                        "key": ix,
                        "type": "dummy",
                    },
                )
            )
            for ix in range(40)
        ]
    )

    await asyncio.sleep(3)

    return


@pytest.mark.skip(reason="Very flaky test")
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        (
            ["poetry", "run", "python", "examples/priority/worker.py", "--slots", "1"],
            8003,
        )
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_priority(
    hatchet: Hatchet, dummy_runs: None, on_demand_worker: Popen[Any]
) -> None:
    test_run_id = str(uuid4())
    choices: list[Priority] = ["low", "medium", "high", "default"]
    N = 30

    run_refs = await priority_workflow.aio_run_many_no_wait(
        [
            priority_workflow.create_bulk_run_item(
                options=TriggerWorkflowOptions(
                    priority=(priority_to_int(pr)),
                    additional_metadata={
                        "priority": pr,
                        "key": ix,
                        "test_run_id": test_run_id,
                    },
                )
            )
            for ix in range(N)
            for pr in [choice(choices)]
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
        limit=1_000,
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
    assert len(runs_ids_started_ats) == N

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


@pytest.mark.skip(reason="Very flaky test")
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        (
            ["poetry", "run", "python", "examples/priority/worker.py", "--slots", "1"],
            8003,
        )
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_priority_via_scheduling(
    hatchet: Hatchet, dummy_runs: None, on_demand_worker: Popen[Any]
) -> None:
    test_run_id = str(uuid4())
    sleep_time = 3
    n = 30
    choices: list[Priority] = ["low", "medium", "high", "default"]
    run_at = datetime.now(tz=timezone.utc) + timedelta(seconds=sleep_time)

    versions = await asyncio.gather(
        *[
            priority_workflow.aio_schedule(
                run_at=run_at,
                options=ScheduleTriggerWorkflowOptions(
                    priority=(priority_to_int(pr)),
                    additional_metadata={
                        "priority": pr,
                        "key": ix,
                        "test_run_id": test_run_id,
                    },
                ),
            )
            for ix in range(n)
            for pr in [choice(choices)]
        ]
    )

    await asyncio.sleep(sleep_time * 2)

    workflow_id = versions[0].workflow_id

    attempts = 0

    while True:
        if attempts >= SLEEP_TIME * n * 2:
            raise TimeoutError("Timed out waiting for runs to finish")

        attempts += 1
        await asyncio.sleep(1)
        runs = await hatchet.runs.aio_list(
            workflow_ids=[workflow_id],
            additional_metadata={
                "test_run_id": test_run_id,
            },
            limit=1_000,
        )

        if not runs.rows:
            continue

        if any(
            r.status in [V1TaskStatus.FAILED, V1TaskStatus.CANCELLED] for r in runs.rows
        ):
            raise ValueError("One or more runs failed or were cancelled")

        if all(r.status == V1TaskStatus.COMPLETED for r in runs.rows):
            break

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


@pytest_asyncio.fixture(loop_scope="session", scope="function")
async def crons(
    hatchet: Hatchet, dummy_runs: None
) -> AsyncGenerator[tuple[str, str, int], None]:
    test_run_id = str(uuid4())
    choices: list[Priority] = ["low", "medium", "high"]
    n = 30

    crons = await asyncio.gather(
        *[
            hatchet.cron.aio_create(
                workflow_name=priority_workflow.name,
                cron_name=f"{test_run_id}-cron-{i}",
                expression="* * * * *",
                input={},
                additional_metadata={
                    "trigger": "cron",
                    "test_run_id": test_run_id,
                    "priority": (priority := choice(choices)),
                    "key": str(i),
                },
                priority=(priority_to_int(priority)),
            )
            for i in range(n)
        ]
    )

    yield crons[0].workflow_id, test_run_id, n

    await asyncio.gather(*[hatchet.cron.aio_delete(cron.metadata.id) for cron in crons])


def time_until_next_minute() -> float:
    now = datetime.now(tz=timezone.utc)
    next_minute = (now + timedelta(minutes=1)).replace(second=0, microsecond=0)

    return (next_minute - now).total_seconds()


@pytest.mark.skip(
    reason="Test is flaky because the first jobs that are picked up don't necessarily go in priority order"
)
@pytest.mark.parametrize(
    "on_demand_worker",
    [
        (
            ["poetry", "run", "python", "examples/priority/worker.py", "--slots", "1"],
            8003,
        )
    ],
    indirect=True,
)
@pytest.mark.asyncio(loop_scope="session")
async def test_priority_via_cron(
    hatchet: Hatchet, crons: tuple[str, str, int], on_demand_worker: Popen[Any]
) -> None:
    workflow_id, test_run_id, n = crons

    await asyncio.sleep(time_until_next_minute() + 10)

    attempts = 0

    while True:
        if attempts >= SLEEP_TIME * n * 2:
            raise TimeoutError("Timed out waiting for runs to finish")

        attempts += 1
        await asyncio.sleep(1)
        runs = await hatchet.runs.aio_list(
            workflow_ids=[workflow_id],
            additional_metadata={
                "test_run_id": test_run_id,
            },
            limit=1_000,
        )

        if not runs.rows:
            continue

        if any(
            r.status in [V1TaskStatus.FAILED, V1TaskStatus.CANCELLED] for r in runs.rows
        ):
            raise ValueError("One or more runs failed or were cancelled")

        if all(r.status == V1TaskStatus.COMPLETED for r in runs.rows):
            break

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

    assert len(runs_ids_started_ats) == n

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
