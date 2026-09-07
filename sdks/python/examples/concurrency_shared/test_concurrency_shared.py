import asyncio
from uuid import uuid4

import pytest

from examples.concurrency_shared.worker import (
    RunWindow,
    WorkflowInput,
    task_a,
    task_b,
    task_chain,
    task_mixed,
)
from hatchet_sdk import Hatchet, V1TaskStatus
from hatchet_sdk.runnables.workflow import Standalone

TOLERANCE_MS = 100

ConcurrencyTask = Standalone[WorkflowInput, RunWindow]


async def gather_run_windows(
    runs: list[tuple[ConcurrencyTask, WorkflowInput]],
) -> list[RunWindow]:
    """Trigger each (task, input) concurrently and return the run windows reported by the
    task functions."""
    return list(await asyncio.gather(*(task.aio_run(inp) for task, inp in runs)))


def assert_serialized(windows: list[RunWindow]) -> None:
    """No two run windows may overlap (max concurrency of 1)."""
    ordered = sorted(windows, key=lambda w: w.start_ms)

    for prev, cur in zip(ordered, ordered[1:]):
        assert (
            cur.start_ms >= prev.end_ms - TOLERANCE_MS
        ), f"runs overlapped: {prev} vs {cur}"


def assert_some_overlap(windows: list[RunWindow]) -> None:
    """At least one pair of windows must overlap, proving the worker can run these tasks
    concurrently when no limit binds."""
    for i, a in enumerate(windows):
        for b in windows[i + 1 :]:
            if a.start_ms < b.end_ms and b.start_ms < a.end_ms:
                return

    raise AssertionError(f"expected at least one overlapping pair, got: {windows}")


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_cross_workflow(hatchet: Hatchet) -> None:
    """Runs of two DIFFERENT tasks referencing the same shared strategy (max=1) with the
    same group key never run concurrently."""
    group = f"xwf-{uuid4()}"

    windows = await gather_run_windows(
        [
            (task_a, WorkflowInput(group=group)),
            (task_a, WorkflowInput(group=group)),
            (task_b, WorkflowInput(group=group)),
            (task_b, WorkflowInput(group=group)),
        ]
    )

    assert_serialized(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_mixed_shared_limit_binds(hatchet: Hatchet) -> None:
    """The mixed task holds an inline strategy AND the shared strategy. With distinct
    inline keys but one shared group key, the shared limit serializes the runs — including
    against a different task."""
    group = f"mixed-shared-{uuid4()}"

    windows = await gather_run_windows(
        [
            (task_mixed, WorkflowInput(group=group, inline=f"inline-{uuid4()}")),
            (task_mixed, WorkflowInput(group=group, inline=f"inline-{uuid4()}")),
            (task_a, WorkflowInput(group=group)),
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
            (task_mixed, WorkflowInput(group=f"group-{uuid4()}", inline=inline)),
            (task_mixed, WorkflowInput(group=f"group-{uuid4()}", inline=inline)),
        ]
    )

    assert_serialized(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_no_limit_overlaps(hatchet: Hatchet) -> None:
    """Control: with no colliding keys, runs overlap freely, proving the serialization in
    the other tests comes from the concurrency strategies rather than worker capacity.
    """
    windows = await gather_run_windows(
        [
            (
                task_mixed,
                WorkflowInput(group=f"group-{uuid4()}", inline=f"inline-{uuid4()}"),
            ),
            (
                task_mixed,
                WorkflowInput(group=f"group-{uuid4()}", inline=f"inline-{uuid4()}"),
            ),
        ]
    )

    assert_some_overlap(windows)


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_concurrency_multi_strategy_chain(hatchet: Hatchet) -> None:
    """A chain mixing two tenant-scoped GROUP_ROUND_ROBIN strategies around a
    workflow-scoped CANCEL_IN_PROGRESS strategy applies each entry's own semantics: two
    runs with distinct tenant keys but a colliding CANCEL_IN_PROGRESS key must resolve by
    the newer run cancelling the older in-flight one."""
    inline = f"chain-inline-{uuid4()}"

    ref_old = await task_chain.aio_run(
        WorkflowInput(group=f"a-{uuid4()}", inline=inline, chain_c=f"c-{uuid4()}"),
        wait_for_result=False,
    )

    # let the first run acquire its full chain and start executing
    await asyncio.sleep(3)

    ref_new = await task_chain.aio_run(
        WorkflowInput(group=f"a-{uuid4()}", inline=inline, chain_c=f"c-{uuid4()}"),
        wait_for_result=False,
    )

    # the newer run completes; the older one is cancelled mid-run by CANCEL_IN_PROGRESS
    await ref_new.aio_result()

    # wait for the OLAP repo to catch up, then assert statuses
    old_run = await hatchet.runs.aio_get(ref_old.workflow_run_id)
    new_run = await hatchet.runs.aio_get(ref_new.workflow_run_id)

    for _ in range(20):
        await asyncio.sleep(1)

        old_run = await hatchet.runs.aio_get(ref_old.workflow_run_id)
        new_run = await hatchet.runs.aio_get(ref_new.workflow_run_id)

        if (
            old_run.run.status == V1TaskStatus.CANCELLED
            and new_run.run.status == V1TaskStatus.COMPLETED
        ):
            break

    assert old_run.run.status == V1TaskStatus.CANCELLED, old_run.run.status
    assert new_run.run.status == V1TaskStatus.COMPLETED, new_run.run.status
