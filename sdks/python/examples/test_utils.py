import asyncio
import time
from collections.abc import Awaitable, Callable
from datetime import datetime
from typing import Any, TypeVar

from hatchet_sdk import Event, Hatchet, RunStatus, TaskRunSummary
from hatchet_sdk.clients.admin import WorkflowRunDetail
from hatchet_sdk.clients.rest.models.v1_workflow_run_details import V1WorkflowRunDetails
from hatchet_sdk.clients.rest.models.worker_status import WorkerStatus

T = TypeVar("T")

TERMINAL_RUN_STATUSES: list[RunStatus] = ["COMPLETED", "FAILED", "CANCELLED"]


async def poll_until(
    fetch: Callable[[], Awaitable[T]],
    is_satisfied: Callable[[T], bool],
    *,
    description: str,
    timeout: float = 60.0,
    interval: float = 0.5,
) -> T:
    deadline = time.monotonic() + timeout

    while True:
        value = await fetch()

        if is_satisfied(value):
            return value

        if time.monotonic() >= deadline:
            raise TimeoutError(
                f"timed out after {timeout}s waiting for {description}; last value: {value!r}"
            )

        await asyncio.sleep(interval)


async def wait_for_run_details(
    hatchet: Hatchet,
    run_id: str,
    is_satisfied: Callable[[WorkflowRunDetail], bool],
    *,
    description: str,
    timeout: float = 60.0,
    interval: float = 0.5,
) -> WorkflowRunDetail:
    return await poll_until(
        lambda: hatchet.runs.aio_get_details(run_id),
        is_satisfied,
        description=f"run {run_id} {description}",
        timeout=timeout,
        interval=interval,
    )


async def wait_for_run_status(
    hatchet: Hatchet, run_id: str, status: RunStatus, timeout: float = 60.0
) -> WorkflowRunDetail:
    return await wait_for_run_details(
        hatchet,
        run_id,
        lambda details: details.status == status,
        description=f"to reach {status}",
        timeout=timeout,
    )


async def wait_for_run_done(
    hatchet: Hatchet, run_id: str, timeout: float = 60.0
) -> WorkflowRunDetail:
    return await wait_for_run_details(
        hatchet,
        run_id,
        lambda details: details.done,
        description="to finish",
        timeout=timeout,
    )


async def wait_for_task_status(
    hatchet: Hatchet,
    run_id: str,
    task_name: str,
    status: RunStatus,
    timeout: float = 60.0,
) -> WorkflowRunDetail:
    def task_has_status(details: WorkflowRunDetail) -> bool:
        task_run = details.task_runs.get(task_name)
        return task_run is not None and task_run.status == status

    return await wait_for_run_details(
        hatchet,
        run_id,
        task_has_status,
        description=f"task {task_name} to reach {status}",
        timeout=timeout,
    )


async def wait_for_task_completed_in_read_model(
    hatchet: Hatchet, run_id: str, task_name: str, timeout: float = 60.0
) -> V1WorkflowRunDetails:
    # a task's display name is "<step readable id>-<unix timestamp>"
    def task_is_completed(details: V1WorkflowRunDetails) -> bool:
        return any(
            t.status == "COMPLETED" and t.display_name.startswith(f"{task_name}-")
            for t in details.tasks
        )

    return await poll_until(
        lambda: hatchet.runs.aio_get(run_id),
        task_is_completed,
        description=f"task {task_name} of run {run_id} to complete in the read model",
        timeout=timeout,
    )


async def wait_for_running_status(
    hatchet: Hatchet,
    run_id: str,
    timeout: float = 60.0,
    min_task_runs: int = 0,
) -> None:
    # A run can go RUNNING -> COMPLETED between two polls, so callers treat
    # this as a best-effort wait rather than an assertion.
    try:
        await wait_for_run_details(
            hatchet,
            run_id,
            lambda details: details.status == "RUNNING"
            and len(details.task_runs) >= min_task_runs,
            description="to start running",
            timeout=timeout,
        )
    except TimeoutError:
        return


async def wait_for_replay(hatchet: Hatchet, run_id: str, timeout: float = 30.0) -> None:
    # A fast replay can finish again between two polls, so callers treat this
    # as a best-effort wait rather than an assertion.
    try:
        await wait_for_run_details(
            hatchet,
            run_id,
            lambda details: details.status != "COMPLETED",
            description="to leave COMPLETED after a replay",
            timeout=timeout,
            interval=0.25,
        )
    except TimeoutError:
        return


async def wait_for_worker_status(
    hatchet: Hatchet,
    worker_name: str,
    status: WorkerStatus,
    timeout: float = 30.0,
) -> None:
    namespaced_worker_name = hatchet.config.apply_namespace(worker_name)

    await poll_until(
        hatchet.workers.aio_list,
        lambda workers: any(
            w.name == namespaced_worker_name and w.status == status for w in workers
        ),
        description=f"worker {worker_name} to report {status.value}",
        timeout=timeout,
        interval=1.0,
    )


async def poll_for_runs(
    hatchet: Hatchet,
    *,
    expected_count: int,
    additional_metadata: dict[str, str] | None = None,
    workflow_ids: list[str] | None = None,
    parent_task_run_id: str | None = None,
    triggering_event_id: str | None = None,
    statuses: list[RunStatus] | None = None,
    timeout: float = 60.0,
    interval: float = 0.5,
) -> list[TaskRunSummary]:
    accepted_statuses = statuses or TERMINAL_RUN_STATUSES

    return await poll_until(
        lambda: hatchet.runs.aio_list(
            additional_metadata=additional_metadata,
            workflow_ids=workflow_ids,
            parent_task_run_id=parent_task_run_id,
            triggering_event_id=triggering_event_id,
            limit=1000,
        ),
        lambda runs: len(runs) >= expected_count
        and all(r.status in accepted_statuses for r in runs),
        description=f"at least {expected_count} runs in {accepted_statuses}",
        timeout=timeout,
        interval=interval,
    )


async def wait_for_child_runs(
    hatchet: Hatchet,
    parent_task_run_id: str,
    min_count: int = 1,
    timeout: float = 30.0,
    **list_kwargs: Any,
) -> list[TaskRunSummary]:
    return await poll_until(
        lambda: hatchet.runs.aio_list(
            parent_task_run_id=parent_task_run_id, **list_kwargs
        ),
        lambda runs: len(runs) >= min_count,
        description=f"at least {min_count} children of {parent_task_run_id}",
        timeout=timeout,
    )


async def wait_for_event(
    hatchet: Hatchet,
    webhook_name: str,
    test_start: datetime,
) -> Event:
    async def find_event_triggered_by_webhook() -> Event | None:
        events = await hatchet.events.aio_list(since=test_start)
        return next(
            (e for e in events if e.triggering_webhook_name == webhook_name), None
        )

    event = await poll_until(
        find_event_triggered_by_webhook,
        lambda event: event is not None,
        description=f"an event triggered by webhook {webhook_name}",
        timeout=30.0,
        interval=1.0,
    )
    assert event is not None
    return event


async def wait_for_workflow_run(
    hatchet: Hatchet, event_id: str, test_start: datetime
) -> TaskRunSummary:
    async def find_completed_run_for_event() -> TaskRunSummary | None:
        runs = await hatchet.runs.aio_list(
            since=test_start,
            additional_metadata={"hatchet__event_id": event_id},
        )
        return next((r for r in runs if r.status == "COMPLETED"), None)

    run = await poll_until(
        find_completed_run_for_event,
        lambda run: run is not None,
        description=f"a completed run triggered by event {event_id}",
        timeout=45.0,
        interval=1.0,
    )
    assert run is not None
    return run
