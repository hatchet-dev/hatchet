import asyncio
import multiprocessing
import threading
from typing import Any
from unittest.mock import MagicMock

import pytest

from hatchet_sdk import Context, EmptyModel, Hatchet
from hatchet_sdk.runnables.action import (
    Action,
    ActionPayload,
    ActionType,
    BatchItemData,
)
from hatchet_sdk.runnables.types import BatchMemberId
from hatchet_sdk.worker.runner.runner import Runner
from hatchet_sdk.worker.slot_usage import (
    create_shared_slot_usage_by_pool,
    read_slot_usage,
)


def _make_runner(hatchet: Hatchet, task: Any) -> Runner:
    runner = Runner(
        event_queue=MagicMock(),
        config=hatchet.config,
        slots=10,
        durable_slots=0,
        handle_kill=False,
        action_registry={task.name: task},
        labels=[],
        lifespan_context=None,
        log_sender=MagicMock(),
        shared_slot_usage_by_pool=create_shared_slot_usage_by_pool(
            multiprocessing.get_context("spawn"), {"default": 10}
        ),
    )
    runner.dispatcher_client = MagicMock()

    return runner


def _make_action(task: Any, action_type: ActionType, **batch_fields: Any) -> Action:
    return Action(
        worker_id="worker-id",
        tenant_id="tenant-id",
        workflow_run_id="workflow-run-id",
        job_id="job-id",
        job_name="job-name",
        job_run_id="job-run-id",
        step_id="step-id",
        step_run_id="step-run-id",
        action_id=task.name,
        action_type=action_type,
        retry_count=0,
        action_payload=ActionPayload(),
        **batch_fields,
    )


async def _wait_until_used_default_slots(runner: Runner, expected: int) -> None:
    async with asyncio.timeout(10):
        # Polls shared memory written by the runner, so there is no asyncio.Event to wait on.
        while read_slot_usage(runner.shared_slot_usage_by_pool)["default"] != expected:
            await asyncio.sleep(0.01)  # noqa: ASYNC110


@pytest.mark.asyncio(loop_scope="session")
async def test_task_stops_being_counted_when_it_releases_its_slot(
    hatchet: Hatchet,
) -> None:
    may_release = threading.Event()
    may_finish = threading.Event()

    def releases_slot_midway(input: EmptyModel, ctx: Context) -> None:
        may_release.wait(timeout=10)
        ctx.release_slot()
        may_finish.wait(timeout=10)

    task = hatchet.task(name="releases-slot", slot_cost=3)(releases_slot_midway)._task
    runner = _make_runner(hatchet, task)

    run = asyncio.create_task(
        runner.handle_start_step_run(_make_action(task, ActionType.START_STEP_RUN))
    )

    await _wait_until_used_default_slots(runner, 3)

    may_release.set()
    await _wait_until_used_default_slots(runner, 0)
    assert not run.done()

    may_finish.set()
    await run
    await _wait_until_used_default_slots(runner, 0)


@pytest.mark.asyncio(loop_scope="session")
async def test_running_batch_is_counted_as_one_default_slot(hatchet: Hatchet) -> None:
    may_finish = asyncio.Event()

    @hatchet.workflow(name="batch-slot-counting").batch_task(batch_max_size=5)
    async def batched(
        tasks: dict[BatchMemberId, EmptyModel], ctx: Context
    ) -> dict[BatchMemberId, dict[str, str]]:
        await may_finish.wait()
        return {member_id: {} for member_id in tasks}

    runner = _make_runner(hatchet, batched)
    batch_items = {
        BatchMemberId(f"member-{index}"): BatchItemData(
            payload=ActionPayload(), workflow_run_id="workflow-run-id"
        )
        for index in range(5)
    }

    run = asyncio.create_task(
        runner.handle_start_batch(
            _make_action(
                batched,
                ActionType.START_BATCH,
                batch_id="batch-id",
                batch_items=batch_items,
            )
        )
    )

    await _wait_until_used_default_slots(runner, 1)

    may_finish.set()
    await run
    await _wait_until_used_default_slots(runner, 0)
