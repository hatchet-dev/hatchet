import pytest
import pytest_asyncio

from hatchet_sdk import Hatchet, RunStatus
from datetime import timedelta, datetime, timezone

from examples.workflow_pause.worker import pausable_workflow
import asyncio
import contextlib
import fcntl
import tempfile
import time
from collections.abc import AsyncGenerator
from pathlib import Path
from uuid import uuid4

# Interval at which the engine's ticker reloads user cron schedules.
CRON_RELOAD_SECONDS = 15

# Every test in this module pauses and unpauses the shared `pausable_workflow`,
# so two of them running at the same time in different pytest-xdist processes
# corrupt each other: a pause with a one second queue TTL in one test cancels the
# runs another test is waiting on, and an unpause in one test lets runs through
# that another test expects to stay queued. An exclusive file lock serializes the
# tests across processes without changing how the suite is invoked.
PAUSE_TEST_LOCK = Path(tempfile.gettempdir()) / "hatchet-python-sdk-workflow-pause.lock"

# The engine caches a workflow's paused flag on the trigger path for five seconds,
# so a pause issued right after an unpause is not honored for new runs until the
# cached entry expires.
PAUSE_FLAG_CACHE_TTL_SECONDS = 6


@pytest_asyncio.fixture(autouse=True, loop_scope="session")
async def serialize_workflow_pause_tests() -> AsyncGenerator[None, None]:
    with PAUSE_TEST_LOCK.open("w") as lock_file:
        fcntl.flock(lock_file, fcntl.LOCK_EX)
        try:
            yield
        finally:
            # Leave the workflow unpaused for whichever test runs next, and let the
            # cached paused flag expire before handing over the lock so that test's
            # own pause takes effect immediately.
            with contextlib.suppress(Exception):
                await pausable_workflow.aio_unpause()
            await asyncio.sleep(PAUSE_FLAG_CACHE_TTL_SECONDS)
            fcntl.flock(lock_file, fcntl.LOCK_UN)


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_pause_cancel_after_ttl(hatchet: Hatchet) -> None:
    await pausable_workflow.aio_pause(
        queue_ttl=timedelta(seconds=1),
    )

    ref = await pausable_workflow.aio_run(wait_for_result=False)

    start_time = time.time()
    run_id = ref.workflow_run_id
    timeout = 10  # seconds

    while True:
        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == RunStatus.CANCELLED:
            return

        assert details.status == RunStatus.QUEUED, (
            f"Run {run_id} is not queued (status {details.status})."
        )

        if time.time() - start_time > timeout:
            assert False, f"Run {run_id} was not cancelled within {timeout} seconds."

        await asyncio.sleep(1)


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_unpause(hatchet: Hatchet) -> None:
    await pausable_workflow.aio_pause(
        queue_ttl=timedelta(minutes=10),
    )

    ref = await pausable_workflow.aio_run(wait_for_result=False)
    run_id = ref.workflow_run_id

    for _ in range(3):
        details = await hatchet.runs.aio_get_details(run_id)

        assert details.status == RunStatus.QUEUED, (
            f"Run {run_id} is not queued (status {details.status})."
        )

        await asyncio.sleep(1)

    await pausable_workflow.aio_unpause()

    timeout = 60  # seconds - this can take a while because we rely on polling internally to re-queue
    start = time.time()

    while True:
        if time.time() - start > timeout:
            assert False, f"Run {run_id} was not completed within {timeout} seconds."

        details = await hatchet.runs.aio_get_details(run_id)

        if details.status == RunStatus.CANCELLED:
            assert False, f"Run {run_id} was cancelled after unpausing."

        if details.status == RunStatus.COMPLETED:
            return

        assert details.status in [
            RunStatus.QUEUED,
            RunStatus.RUNNING,
        ], f"Run {run_id} is not queued or running (status {details.status})."

        await asyncio.sleep(1)


async def trigger_workflows(test_run_id: str, runs: set[str]) -> None:
    while True:
        ref = await pausable_workflow.aio_run(
            wait_for_result=False,
            additional_metadata={"test_run_id": test_run_id},
        )
        runs.add(ref.workflow_run_id)
        await asyncio.sleep(0.125)


async def wait_for_all_completed(
    hatchet: Hatchet, run_ids: set[str], timeout: float
) -> dict[str, RunStatus]:
    # Returns the runs that did not complete before the deadline, keyed by run id
    # with the last status observed for each. Stops early if a run reaches a
    # terminal status other than COMPLETED, since it can never complete after that.
    deadline = time.time() + timeout
    remaining: dict[str, RunStatus] = dict.fromkeys(run_ids, RunStatus.QUEUED)
    terminal = {RunStatus.CANCELLED, RunStatus.FAILED}

    while remaining and time.time() < deadline:
        details = await asyncio.gather(
            *[hatchet.runs.aio_get_details(run_id) for run_id in remaining]
        )

        for run_id, detail in zip(list(remaining), details):
            if detail.status == RunStatus.COMPLETED:
                del remaining[run_id]
            else:
                remaining[run_id] = detail.status

        if any(status in terminal for status in remaining.values()):
            break

        await asyncio.sleep(1)

    return remaining


@pytest.mark.asyncio(loop_scope="session")
async def test_unpause_no_stranded_runs(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())
    run_ids = set[str]()

    t = asyncio.create_task(trigger_workflows(test_run_id, run_ids))

    await asyncio.sleep(3)

    await pausable_workflow.aio_pause(
        queue_ttl=timedelta(minutes=10),
    )

    await asyncio.sleep(3)

    await pausable_workflow.aio_unpause()

    await asyncio.sleep(3)

    t.cancel()
    with contextlib.suppress(asyncio.CancelledError):
        await t

    # Runs triggered before the pause, while paused, and shortly after the unpause
    # (the engine caches the paused flag on the trigger path for a few seconds)
    # must all complete. Runs that landed in the paused table after the unpause
    # are requeued by a periodic reconciliation, so this needs the same generous
    # window as test_workflow_unpause rather than a fixed few seconds.
    stranded = await wait_for_all_completed(hatchet, run_ids, timeout=60)

    assert not stranded, (
        f"{len(stranded)} of {len(run_ids)} runs did not complete after unpausing: {stranded}"
    )


@pytest.mark.asyncio(loop_scope="session")
async def test_workflow_pause_drop_crons_and_schedules(hatchet: Hatchet) -> None:
    test_run_id = str(uuid4())

    await pausable_workflow.aio_pause(
        queue_ttl=timedelta(minutes=1),
        paused_workflow_scheduled_run_queue_behavior="DROP",
        paused_workflow_cron_run_queue_behavior="DROP",
    )

    cron = await pausable_workflow.aio_create_cron(
        cron_name=test_run_id + "_cron",
        expression="* * * * * *",
        additional_metadata={
            "test_run_id": test_run_id,
        },
    )

    # The cron fires every second. If it outlives this test, it triggers the
    # workflow continuously for the rest of the session and keeps the engine's
    # cached paused flag fresh, which makes every later pause in this file
    # invisible for a few seconds. Always delete it, even when an assertion fails.
    try:
        await pausable_workflow.aio_schedule(
            run_at=datetime.now(timezone.utc) + timedelta(seconds=1),
            additional_metadata={
                "test_run_id": test_run_id,
            },
        )

        for _ in range(10):
            runs = await pausable_workflow.aio_list_runs(
                additional_metadata={"test_run_id": test_run_id}
            )

            assert len(runs) == 0, f"Expected no runs, got {len(runs)}"

            await asyncio.sleep(1)
    finally:
        await hatchet.crons.aio_delete(cron.metadata.id)
        # The engine reloads user crons every 15 seconds, so the deleted cron can
        # keep firing until the next reload. Stay paused (its runs are dropped)
        # until it has stopped, otherwise the late fires keep refreshing the
        # cached paused flag and the next test's pause is not honored.
        await asyncio.sleep(CRON_RELOAD_SECONDS + 2)
        await pausable_workflow.aio_unpause()
