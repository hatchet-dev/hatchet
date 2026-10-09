import asyncio
from collections import defaultdict
from collections.abc import Callable
from datetime import datetime
from random import choice
from typing import Literal
from uuid import uuid4

import pytest
from pydantic import BaseModel

from examples.concurrency_workflow_level.worker import (
    DIGIT_MAX_RUNS,
    NAME_MAX_RUNS,
    WorkflowInput,
    concurrency_workflow_level_workflow,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.models.v1_task_summary import V1TaskSummary

Character = Literal["Anna", "Vronsky", "Stiva", "Dolly", "Levin", "Karenin"]
characters: list[Character] = [
    "Anna",
    "Vronsky",
    "Stiva",
    "Dolly",
    "Levin",
    "Karenin",
]

NUM_RUNS = 100
TASKS_PER_RUN = 2


class RunWindow(BaseModel):
    run_id: str
    name: str
    digit: str
    started_at: datetime
    finished_at: datetime


def build_run_windows(tasks: list[V1TaskSummary]) -> list[RunWindow]:
    """Build one execution window per workflow run from its task rows.

    A workflow run holds its workflow-level concurrency slots from the moment
    its first task starts until its last task finishes, so its window is the
    union of its task windows.

    The windows are built from task rows (only_tasks=True) on purpose: task
    started_at/finished_at come from the STARTED/FINISHED event timestamps
    stamped by the worker, so a run that only starts after another run
    released a slot always has a later started_at than that run's finished_at.
    DAG rows instead derive these timestamps from the OLAP write time of the
    events, which lags and reorders under load and makes sequential handoffs
    look like overlaps.

    :param tasks: The task rows of every run under test.
    :return: One window per workflow run.
    """
    windows: dict[str, RunWindow] = {}

    for task in tasks:
        assert (
            task.started_at is not None
        ), f"task {task.task_external_id} has no started_at"
        assert (
            task.finished_at is not None
        ), f"task {task.task_external_id} has no finished_at"

        meta = task.additional_metadata or {}
        existing = windows.get(task.workflow_run_external_id)

        if existing is not None:
            existing.started_at = min(existing.started_at, task.started_at)
            existing.finished_at = max(existing.finished_at, task.finished_at)
        else:
            windows[task.workflow_run_external_id] = RunWindow(
                run_id=task.workflow_run_external_id,
                name=meta.get("name", ""),
                digit=meta.get("digit", ""),
                started_at=task.started_at,
                finished_at=task.finished_at,
            )

    return list(windows.values())


def peak_concurrency_by_key(
    windows: list[RunWindow], key_of: Callable[[RunWindow], str]
) -> dict[str, int]:
    """Sweep over the start and finish instants of the windows.

    A window that starts at the exact instant another one finishes is a
    handoff, not an overlap.

    :param windows: The run windows to sweep over.
    :param key_of: Extracts the concurrency key of a window.
    :return: The peak number of windows open at the same time, per key.
    """
    by_key: dict[str, list[RunWindow]] = defaultdict(list)

    for window in windows:
        by_key[key_of(window)].append(window)

    peaks: dict[str, int] = {}

    for key, key_windows in by_key.items():
        events: list[tuple[datetime, int]] = []
        for window in key_windows:
            events.append((window.started_at, 1))
            events.append((window.finished_at, -1))
        events.sort()

        open_windows = 0
        peak = 0
        for _, delta in events:
            open_windows += delta
            peak = max(peak, open_windows)
        peaks[key] = peak

    return peaks


def violations(peaks: dict[str, int], limit: int) -> dict[str, int]:
    return {key: peak for key, peak in peaks.items() if peak > limit}


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_level_concurrency(hatchet: Hatchet, test_run_id: str) -> None:
    run_refs = await concurrency_workflow_level_workflow.aio_run_many(
        [
            concurrency_workflow_level_workflow.create_bulk_run_item(
                WorkflowInput(
                    name=(name := choice(characters)),
                    digit=(digit := choice([str(i) for i in range(6)])),
                ),
                additional_metadata={
                    "test_run_id": test_run_id,
                    "key": f"{name}-{digit}",
                    "name": name,
                    "digit": digit,
                },
            )
            for _ in range(NUM_RUNS)
        ],
        wait_for_result=False,
    )

    await asyncio.sleep(10)
    await asyncio.gather(*[r.aio_result() for r in run_refs])

    workflows = await hatchet.workflows.aio_list(
        workflow_name=concurrency_workflow_level_workflow.name,
        limit=1_000,
    )

    assert workflows

    workflow = next(
        (w for w in workflows if w.name == concurrency_workflow_level_workflow.name),
        None,
    )

    assert workflow

    assert workflow.name == concurrency_workflow_level_workflow.name

    # The OLAP rows are written asynchronously, so wait until every task of
    # every run has both timestamps before measuring.
    tasks: list[V1TaskSummary] = []
    deadline = asyncio.get_running_loop().time() + 60

    while True:
        tasks = await hatchet.runs.aio_list(
            workflow_ids=[workflow.metadata.id],
            additional_metadata={
                "test_run_id": test_run_id,
            },
            limit=1_000,
            only_tasks=True,
        )

        if len(tasks) == NUM_RUNS * TASKS_PER_RUN and all(
            t.started_at is not None and t.finished_at is not None for t in tasks
        ):
            break

        assert (
            asyncio.get_running_loop().time() < deadline
        ), f"timed out waiting for task rows with timestamps, have {len(tasks)}"

        await asyncio.sleep(0.5)

    windows = build_run_windows(tasks)

    assert len(windows) == NUM_RUNS

    peak_by_digit = peak_concurrency_by_key(windows, lambda w: w.digit)
    peak_by_name = peak_concurrency_by_key(windows, lambda w: w.name)

    assert violations(peak_by_digit, DIGIT_MAX_RUNS) == {}
    assert violations(peak_by_name, NAME_MAX_RUNS) == {}
