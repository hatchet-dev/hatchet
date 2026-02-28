from __future__ import annotations

import asyncio

import pytest

from examples.durable.worker import durable_error_task, durable_sleep_event_spawn
from examples.durable.worker import SLEEP_TIME as DURABLE_SLEEP_TIME
from examples.durable_statuses.worker import status_long_sleep, status_short_sleep
from examples.durable_eviction.worker import evictable_sleep
from hatchet_sdk import Hatchet
from hatchet_sdk.clients.rest.api.task_api import TaskApi
from hatchet_sdk.clients.rest.models.v1_task_status import V1TaskStatus

POLL_INTERVAL = 0.2
MAX_POLLS = 150


async def _poll_rest_status(
    hatchet: Hatchet,
    workflow_run_id: str,
    target: V1TaskStatus,
    max_polls: int = MAX_POLLS,
    interval: float = POLL_INTERVAL,
) -> V1TaskStatus:
    for _ in range(max_polls):
        status = await hatchet.runs.aio_get_status(workflow_run_id)
        if status == target:
            return status
        await asyncio.sleep(interval)
    return await hatchet.runs.aio_get_status(workflow_run_id)


async def _get_task_id_from_details(hatchet: Hatchet, workflow_run_id: str) -> str:
    details = await hatchet.runs.aio_get_details(workflow_run_id)
    return list(details.task_runs.values())[0].external_id


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_queued_to_running(hatchet: Hatchet) -> None:
    ref = status_long_sleep.run_no_wait()

    status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
    assert status in (V1TaskStatus.QUEUED, V1TaskStatus.RUNNING)

    await ref.aio_result()


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_completed(hatchet: Hatchet) -> None:
    ref = status_short_sleep.run_no_wait()

    status = await _poll_rest_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.COMPLETED
    )
    assert status == V1TaskStatus.COMPLETED


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_cancelled(hatchet: Hatchet) -> None:
    ref = durable_sleep_event_spawn.run_no_wait()

    await asyncio.sleep(DURABLE_SLEEP_TIME + 2)
    await hatchet.runs.aio_cancel(ref.workflow_run_id)

    status = await _poll_rest_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.CANCELLED
    )
    assert status == V1TaskStatus.CANCELLED


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_failed(hatchet: Hatchet) -> None:
    ref = durable_error_task.run_no_wait()

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.FAILED)
    assert status == V1TaskStatus.FAILED


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_evicted(hatchet: Hatchet) -> None:
    ref = evictable_sleep.run_no_wait()

    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.RUNNING:
            break
        await asyncio.sleep(POLL_INTERVAL)

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED)
    assert status == V1TaskStatus.EVICTED


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_evicted_then_running(hatchet: Hatchet) -> None:
    ref = evictable_sleep.run_no_wait()

    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.RUNNING:
            break
        await asyncio.sleep(POLL_INTERVAL)

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED)
    assert status == V1TaskStatus.EVICTED

    task_id = await _get_task_id_from_details(hatchet, ref.workflow_run_id)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    assert status == V1TaskStatus.RUNNING

    result = await ref.aio_result()
    assert result["status"] == "completed"


@pytest.mark.asyncio(loop_scope="session")
async def test_api_status_full_lifecycle(hatchet: Hatchet) -> None:
    ref = evictable_sleep.run_no_wait()

    for _ in range(MAX_POLLS):
        status = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        if status == V1TaskStatus.RUNNING:
            break
        await asyncio.sleep(POLL_INTERVAL)

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.EVICTED)
    assert status == V1TaskStatus.EVICTED

    task_id = await _get_task_id_from_details(hatchet, ref.workflow_run_id)
    with hatchet.runs.client() as client:
        TaskApi(client).v1_task_restore(task=task_id)

    status = await _poll_rest_status(hatchet, ref.workflow_run_id, V1TaskStatus.RUNNING)
    assert status == V1TaskStatus.RUNNING

    status = await _poll_rest_status(
        hatchet, ref.workflow_run_id, V1TaskStatus.COMPLETED
    )
    assert status == V1TaskStatus.COMPLETED

    result = await ref.aio_result()
    assert result["status"] == "completed"


@pytest.mark.asyncio(loop_scope="session")
async def test_api_get_vs_get_status_consistency(hatchet: Hatchet) -> None:
    ref = status_short_sleep.run_no_wait()

    for _ in range(MAX_POLLS):
        status_direct = await hatchet.runs.aio_get_status(ref.workflow_run_id)
        run_details = await hatchet.runs.aio_get(ref.workflow_run_id)
        status_via_get = run_details.run.status

        assert (
            status_direct == status_via_get
        ), f"aio_get_status()={status_direct} != aio_get().run.status={status_via_get}"

        if status_direct == V1TaskStatus.COMPLETED:
            break
        await asyncio.sleep(POLL_INTERVAL)
    else:
        pytest.fail("Run did not reach COMPLETED within timeout")
