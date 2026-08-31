import asyncio
from typing import Any
from uuid import uuid4

import pytest

from examples.concurrency_shared.worker import (
    WorkflowInput,
    concurrency_shared_mixed_workflow,
    concurrency_shared_workflow_a,
    concurrency_shared_workflow_b,
)
from hatchet_sdk import Hatchet
from hatchet_sdk.runnables.workflow import Workflow

TOLERANCE_MS = 100


async def gather_run_windows(
    runs: list[tuple[Workflow[WorkflowInput], str, WorkflowInput]],
) -> list[dict[str, int]]:
    """Trigger each (workflow, task_name, input) concurrently and return the run windows
    reported by the task functions."""
    results: list[dict[str, Any]] = await asyncio.gather(
        *(wf.aio_run(inp) for wf, _, inp in runs)
    )

    return [result[task_name] for result, (_, task_name, _) in zip(results, runs)]


def assert_serialized(windows: list[dict[str, int]]) -> None:
    """No two run windows may overlap (max concurrency of 1)."""
    ordered = sorted(windows, key=lambda w: w["start_ms"])

    for prev, cur in zip(ordered, ordered[1:]):
        assert cur["start_ms"] >= prev["end_ms"] - TOLERANCE_MS, (
            f"runs overlapped: {prev} vs {cur}"
        )


def assert_some_overlap(windows: list[dict[str, int]]) -> None:
    """At least one pair of windows must overlap, proving the worker can run these tasks
    concurrently when no limit binds."""
    for i, a in enumerate(windows):
        for b in windows[i + 1 :]:
            if a["start_ms"] < b["end_ms"] and b["start_ms"] < a["end_ms"]:
                return

    raise AssertionError(f"expected at least one overlapping pair, got: {windows}")


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_cross_workflow(hatchet: Hatchet) -> None:
    """Tasks from two DIFFERENT workflows referencing the same shared strategy (max=1)
    with the same group key never run concurrently."""
    group = f"xwf-{uuid4()}"

    windows = await gather_run_windows(
        [
            (concurrency_shared_workflow_a, "task_a", WorkflowInput(group=group)),
            (concurrency_shared_workflow_a, "task_a", WorkflowInput(group=group)),
            (concurrency_shared_workflow_b, "task_b", WorkflowInput(group=group)),
            (concurrency_shared_workflow_b, "task_b", WorkflowInput(group=group)),
        ]
    )

    assert_serialized(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_mixed_shared_limit_binds(hatchet: Hatchet) -> None:
    """The mixed task holds an inline strategy AND the shared strategy. With distinct
    inline keys but one shared group key, the shared limit serializes the runs — including
    against a task from another workflow."""
    group = f"mixed-shared-{uuid4()}"

    windows = await gather_run_windows(
        [
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=group, inline=f"inline-{uuid4()}"),
            ),
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=group, inline=f"inline-{uuid4()}"),
            ),
            (concurrency_shared_workflow_a, "task_a", WorkflowInput(group=group)),
        ]
    )

    assert_serialized(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_mixed_inline_limit_binds(hatchet: Hatchet) -> None:
    """With distinct shared group keys but one inline key, the inline (workflow-scoped)
    strategy serializes the runs — both limits on the mixed task are live at once."""
    inline = f"inline-{uuid4()}"

    windows = await gather_run_windows(
        [
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=f"group-{uuid4()}", inline=inline),
            ),
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=f"group-{uuid4()}", inline=inline),
            ),
        ]
    )

    assert_serialized(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_no_limit_overlaps(hatchet: Hatchet) -> None:
    """Control: with no colliding keys, runs overlap freely, proving the serialization in
    the other tests comes from the concurrency strategies rather than worker capacity."""
    windows = await gather_run_windows(
        [
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=f"group-{uuid4()}", inline=f"inline-{uuid4()}"),
            ),
            (
                concurrency_shared_mixed_workflow,
                "task_mixed",
                WorkflowInput(group=f"group-{uuid4()}", inline=f"inline-{uuid4()}"),
            ),
        ]
    )

    assert_some_overlap(windows)
