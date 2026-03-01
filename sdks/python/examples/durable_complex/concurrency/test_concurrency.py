from __future__ import annotations

import asyncio
from typing import Any
from uuid import uuid4

import pytest

from examples.durable_complex.conftest import get_task_output
from examples.durable_complex.concurrency.worker import (
    ConcurrencyInput,
    durable_concurrency_cancel_in_progress_workflow,
    durable_concurrency_cancel_newest_workflow,
    durable_concurrency_slot_retention_workflow,
    durable_concurrency_workflow,
)
from hatchet_sdk import Hatchet, TriggerWorkflowOptions
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus
from hatchet_sdk.workflow_run import WorkflowRunRef


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_concurrency(hatchet: Hatchet) -> None:
    """Durable task with workflow-level concurrency: 4 runs across 2 groups, all complete.
    Runs are evicted during sleep, restored, then complete."""
    refs: list[WorkflowRunRef] = []
    for group in ("A", "A", "B", "B"):
        ref = await durable_concurrency_workflow.aio_run_no_wait(
            ConcurrencyInput(group=group)
        )
        refs.append(ref)
        await asyncio.sleep(0.3)

    results: list[dict[str, Any]] = await asyncio.wait_for(
        asyncio.gather(*[ref.aio_result() for ref in refs]),
        timeout=60.0,
    )

    assert len(results) == 4
    for r in results:
        out = get_task_output(
            r,
            "durable_concurrency_task",
            "durableconcurrencyworkflow:durable_concurrency_task",
        )
        assert isinstance(out, dict), f"Expected dict output, got {r}"
        assert out.get("status") == "completed", f"Expected status=completed, got {r}"
        assert out.get("group") in ("A", "B"), f"Expected group in (A, B), got {r}"


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_concurrency_cancel_in_progress(hatchet: Hatchet) -> None:
    """CANCEL_IN_PROGRESS: new run cancels the in-progress run; last run completes."""
    test_run_id = str(uuid4())
    refs: list[WorkflowRunRef] = []

    for i in range(5):
        ref = await durable_concurrency_cancel_in_progress_workflow.aio_run_no_wait(
            ConcurrencyInput(group="A"),
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id, "i": str(i)},
            ),
        )
        refs.append(ref)
        await asyncio.sleep(1)

    for ref in refs:
        try:
            await ref.aio_result()
        except Exception:
            pass

    await asyncio.sleep(5)

    runs = sorted(
        hatchet.runs.list(additional_metadata={"test_run_id": test_run_id}).rows,
        key=lambda r: int((r.additional_metadata or {}).get("i", "0")),
    )

    assert len(runs) == 5
    assert (runs[-1].additional_metadata or {}).get("i") == "4"
    assert runs[-1].status == V1TaskStatus.COMPLETED
    assert all(r.status == V1TaskStatus.CANCELLED for r in runs[:-1])


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_concurrency_cancel_newest(hatchet: Hatchet) -> None:
    """CANCEL_NEWEST: first run completes; subsequent runs in same group are cancelled."""
    test_run_id = str(uuid4())
    to_run = await durable_concurrency_cancel_newest_workflow.aio_run_no_wait(
        ConcurrencyInput(group="A"),
        options=TriggerWorkflowOptions(
            additional_metadata={"test_run_id": test_run_id}
        ),
    )
    await asyncio.sleep(1)

    to_cancel = await durable_concurrency_cancel_newest_workflow.aio_run_many_no_wait(
        [
            durable_concurrency_cancel_newest_workflow.create_bulk_run_item(
                input=ConcurrencyInput(group="A"),
                options=TriggerWorkflowOptions(
                    additional_metadata={"test_run_id": test_run_id}
                ),
            )
            for _ in range(5)
        ]
    )

    await to_run.aio_result()

    try:
        await asyncio.wait_for(
            asyncio.gather(
                *[ref.aio_result() for ref in to_cancel], return_exceptions=True
            ),
            timeout=10.0,
        )
    except TimeoutError:
        pass

    await asyncio.sleep(5)

    runs = hatchet.runs.list(additional_metadata={"test_run_id": test_run_id}).rows
    successful_run = next(r for r in runs if r.metadata.id == to_run.workflow_run_id)
    assert successful_run.status == V1TaskStatus.COMPLETED
    assert all(
        r.status == V1TaskStatus.CANCELLED
        for r in runs
        if r.metadata.id != to_run.workflow_run_id
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_durable_concurrency_eviction_holds_slot(hatchet: Hatchet) -> None:
    """Evicted runs must NOT release their concurrency slot.

    With max_runs=1 GROUP_ROUND_ROBIN, start 3 runs for the same group. Tasks
    use durable sleep and get evicted mid-execution. Poll to verify that at no
    point more than max_runs are concurrently RUNNING—proving that evicted runs
    still occupy their concurrency slot.
    """
    max_runs = 1
    num_runs = 3
    test_run_id = str(uuid4())
    refs: list[WorkflowRunRef] = []

    for i in range(num_runs):
        ref = await durable_concurrency_slot_retention_workflow.aio_run_no_wait(
            ConcurrencyInput(group="A"),
            options=TriggerWorkflowOptions(
                additional_metadata={"test_run_id": test_run_id, "i": str(i)},
            ),
        )
        refs.append(ref)
        await asyncio.sleep(0.3)

    max_observed_running = 0
    saw_evicted = False

    for _ in range(120):
        await asyncio.sleep(1)
        runs = hatchet.runs.list(additional_metadata={"test_run_id": test_run_id}).rows
        running_count = sum(1 for r in runs if r.status == V1TaskStatus.RUNNING)
        max_observed_running = max(max_observed_running, running_count)

        if any(r.status == V1TaskStatus.EVICTED for r in runs):
            saw_evicted = True

        if len(runs) == num_runs and all(
            r.status
            in (V1TaskStatus.COMPLETED, V1TaskStatus.FAILED, V1TaskStatus.CANCELLED)
            for r in runs
        ):
            break
    else:
        pytest.fail("Not all runs completed within timeout")

    assert saw_evicted, (
        "No runs were observed in EVICTED status — "
        "test cannot verify slot retention during eviction"
    )

    assert max_observed_running <= max_runs, (
        f"Observed {max_observed_running} concurrent RUNNING runs, "
        f"expected at most {max_runs} — evicted runs released their concurrency slot"
    )

    final_runs = hatchet.runs.list(
        additional_metadata={"test_run_id": test_run_id}
    ).rows
    assert len(final_runs) == num_runs
    assert all(r.status == V1TaskStatus.COMPLETED for r in final_runs)
