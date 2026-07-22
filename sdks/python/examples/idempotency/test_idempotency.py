import pytest

from examples.idempotency.worker import (
    idempotent_task,
    idempotent_task_short_window,
    idempotent_status_based_task,
    idempotent_status_based_task_with_retries,
    IdempotencyInput,
    EVENT_KEY,
)

from hatchet_sdk import (
    Hatchet,
    IdempotencyCollisionError,
    RunStatus,
    BulkTriggerIdempotencyCollisionError,
    FailedTaskRunExceptionGroup,
)
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from uuid import uuid4
from datetime import timedelta, datetime, timezone
import asyncio
from typing import cast
from time import time


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs_direct_trigger(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    ref1 = await idempotent_task.aio_run(
        input=IdempotencyInput(id=test_run_id),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_task.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        break

    assert runs is not None
    assert len(runs.rows) == 1
    assert runs.rows[0].metadata.id == ref1.workflow_run_id

    result = await ref1.aio_result()
    assert "hello" in result["result"].lower()


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based(
    hatchet: Hatchet,
) -> None:
    start = time()
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    res1 = await ref1.aio_result()

    assert time() - start >= 2
    assert (
        time() - start < 10
    ), "The task should have completed within the TTL window so we can test that the status-based idempotency is working."

    ref2 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )
    res2 = await ref2.aio_result()

    assert (
        res1 == res2
    ), "The result of the second run should be the same as the first run."
    assert (
        time() - start >= 4
    ), "The second run should have waited for the first run to complete before returning the result."
    assert (
        time() - start < 10
    ), "The second run should have completed within the TTL window so we can test that the status-based idempotency is working."

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        break

    assert runs is not None
    assert len(runs.rows) == 2
    assert {r.metadata.id for r in runs.rows} == {
        ref1.workflow_run_id,
        ref2.workflow_run_id,
    }

    result1 = await ref1.aio_result()
    assert "hello" in result1["result"].lower()

    assert result1 == res2


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based_failure(
    hatchet: Hatchet,
) -> None:
    start = time()
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="fail"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    with pytest.raises(Exception) as exc_info_2:
        await ref1.aio_result()

    assert "failed as requested" in str(exc_info_2.value).lower()
    assert ref1.workflow_run_id in str(exc_info_2.value).lower()

    assert time() - start >= 2
    assert (
        time() - start < 10
    ), "The task should have finished within the TTL window so we can test that the status-based idempotency is working."

    ref2 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="fail"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(Exception) as exc_info_2:
        await ref2.aio_result()

    assert "failed as requested" in str(exc_info_2.value).lower()
    assert ref2.workflow_run_id in str(exc_info_2.value).lower()

    assert (
        time() - start >= 4
    ), "The second run should have waited for the first run to complete before returning the result."
    assert (
        time() - start < 10
    ), "The second run should have completed within the TTL window so we can test that the status-based idempotency is working."

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        break

    assert runs is not None
    assert len(runs.rows) == 2
    assert {r.metadata.id for r in runs.rows} == {
        ref1.workflow_run_id,
        ref2.workflow_run_id,
    }


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based_cancel(
    hatchet: Hatchet,
) -> None:
    start = time()
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="cancel"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    with pytest.raises(FailedTaskRunExceptionGroup) as exc_info_2:
        await ref1.aio_result()

    assert "task was cancelled" in str(exc_info_2.value).lower()

    details = await hatchet.runs.aio_get_details(ref1.workflow_run_id)

    assert details.status == RunStatus.CANCELLED
    assert details.external_id == ref1.workflow_run_id

    assert time() - start >= 1
    assert (
        time() - start < 10
    ), "The task should have finished within the TTL window so we can test that the status-based idempotency is working."

    ref2 = await idempotent_status_based_task.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="cancel"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(FailedTaskRunExceptionGroup) as exc_info_3:
        await ref2.aio_result()

    assert "task was cancelled" in str(exc_info_3.value).lower()

    details = await hatchet.runs.aio_get_details(ref1.workflow_run_id)

    assert details.status == RunStatus.CANCELLED
    assert details.external_id == ref1.workflow_run_id

    assert (
        time() - start >= 2
    ), "The second run should have waited for the first run to complete before returning the result."
    assert (
        time() - start < 10
    ), "The second run should have completed within the TTL window so we can test that the status-based idempotency is working."

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        break

    assert runs is not None
    assert len(runs.rows) == 2
    assert {r.metadata.id for r in runs.rows} == {
        ref1.workflow_run_id,
        ref2.workflow_run_id,
    }


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs_bulk_trigger(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())

    with pytest.raises(BulkTriggerIdempotencyCollisionError) as exc_info:
        await idempotent_task.aio_run_many(
            [
                idempotent_task.create_bulk_run_item(
                    input=IdempotencyInput(id=test_run_id),
                    additional_metadata={"test_run_id": test_run_id},
                )
                for _ in range(2)
            ],
            wait_for_result=False,
        )

    successes = exc_info.value.successful_workflow_run_external_ids
    collisions = exc_info.value.collisions

    assert len(successes) == 1
    assert len(collisions) == 1


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs_direct_trigger_short_window(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    for i in range(4):
        if i == 1:
            with pytest.raises(IdempotencyCollisionError) as exc_info:
                await idempotent_task_short_window.aio_run(
                    input=IdempotencyInput(id=test_run_id),
                    wait_for_result=False,
                    additional_metadata={"test_run_id": test_run_id},
                )

            assert exc_info.value.existing_run_external_id is not None
        else:
            await idempotent_task_short_window.aio_run(
                input=IdempotencyInput(id=test_run_id),
                wait_for_result=False,
                additional_metadata={"test_run_id": test_run_id},
            )

        ## dynamic sleep, first task should run, second should not, third should, fourth should
        if i != 3:
            await asyncio.sleep(i + 1.5)

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) < 3:
            await asyncio.sleep(1)
            continue

        break
    else:
        pytest.fail("Expected to find at least one run, but found none.")

    assert runs.rows
    assert len(runs.rows) == 3

    for id in [r.metadata.id for r in runs.rows]:
        ref = hatchet.runs.get_run_ref(id)
        res = await ref.aio_result()

        assert "hello" in str(res).lower()


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs_event_trigger(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    e1 = await hatchet.event.aio_push(
        event_key=EVENT_KEY,
        payload={"id": test_run_id},
        additional_metadata={"test_run_id": test_run_id},
    )
    e2 = await hatchet.event.aio_push(
        event_key=EVENT_KEY,
        payload={"id": test_run_id},
        additional_metadata={"test_run_id": test_run_id},
    )

    runs: V1TaskSummaryList | None = None

    for _ in range(15):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if len(runs.rows) == 0:
            await asyncio.sleep(1)
            continue

        break

    assert runs is not None
    assert len(runs.rows) == 1

    await asyncio.sleep(1)

    details = await hatchet.event.aio_list(
        event_ids=[e1.event_id, e2.event_id],
    )

    assert details.rows
    assert len(details.rows) == 2

    all_triggered_runs = [
        *(details.rows[0].triggered_runs or []),
        *(details.rows[1].triggered_runs or []),
    ]

    assert len(all_triggered_runs) == 1

    for _ in range(15):
        run_details = await hatchet.runs.aio_get_details(
            all_triggered_runs[0].workflow_run_id
        )

        if run_details.status in [RunStatus.QUEUED, RunStatus.RUNNING]:
            await asyncio.sleep(1)
            continue

        assert run_details.status == RunStatus.COMPLETED


async def _wait_for_retry_count(
    hatchet: Hatchet, test_run_id: str, min_retry_count: int
) -> None:
    for _ in range(30):
        runs = await hatchet.runs.aio_list(
            since=datetime.now(timezone.utc) - timedelta(minutes=5),
            additional_metadata={"test_run_id": test_run_id},
        )

        if runs.rows and (runs.rows[0].retry_count or 0) >= min_retry_count:
            return

        await asyncio.sleep(1)

    pytest.fail(f"Timed out waiting for run to reach retry_count >= {min_retry_count}.")


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based_key_held_across_all_retries_until_exhausted(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="fail"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    await _wait_for_retry_count(hatchet, test_run_id, min_retry_count=1)

    with pytest.raises(IdempotencyCollisionError) as exc_info_mid_retry:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info_mid_retry.value.existing_run_external_id == ref1.workflow_run_id

    await _wait_for_retry_count(hatchet, test_run_id, min_retry_count=2)

    with pytest.raises(IdempotencyCollisionError) as exc_info_late_retry:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info_late_retry.value.existing_run_external_id == ref1.workflow_run_id

    with pytest.raises(Exception) as exc_info_final:
        await ref1.aio_result()

    assert "failed as requested" in str(exc_info_final.value).lower()

    details = await hatchet.runs.aio_get_details(ref1.workflow_run_id)
    assert details.status == RunStatus.FAILED

    ref2 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="fail"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )
    assert ref2.workflow_run_id != ref1.workflow_run_id

    with pytest.raises(Exception) as exc_info_ref2:
        await ref2.aio_result()

    assert "failed as requested" in str(exc_info_ref2.value).lower()


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based_key_released_immediately_on_success_after_retry(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="success"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    await _wait_for_retry_count(hatchet, test_run_id, min_retry_count=1)

    with pytest.raises(IdempotencyCollisionError) as exc_info_mid_retry:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info_mid_retry.value.existing_run_external_id == ref1.workflow_run_id

    result = await ref1.aio_result()
    assert "hello" in result["result"].lower()

    details = await hatchet.runs.aio_get_details(ref1.workflow_run_id)
    assert details.status == RunStatus.COMPLETED

    ref2 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="success"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )
    assert ref2.workflow_run_id != ref1.workflow_run_id

    result2 = await ref2.aio_result()
    assert "hello" in result2["result"].lower()


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_status_based_key_released_immediately_on_cancel_after_retry(
    hatchet: Hatchet,
) -> None:
    test_run_id = str(uuid4())
    ref1 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="cancel"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id

    await _wait_for_retry_count(hatchet, test_run_id, min_retry_count=1)

    with pytest.raises(IdempotencyCollisionError) as exc_info_mid_retry:
        await idempotent_status_based_task_with_retries.aio_run(
            input=IdempotencyInput(id=test_run_id), wait_for_result=False
        )

    assert exc_info_mid_retry.value.existing_run_external_id == ref1.workflow_run_id

    with pytest.raises(FailedTaskRunExceptionGroup) as exc_info_final:
        await ref1.aio_result()

    assert "task was cancelled" in str(exc_info_final.value).lower()

    details = await hatchet.runs.aio_get_details(ref1.workflow_run_id)
    assert details.status == RunStatus.CANCELLED

    ref2 = await idempotent_status_based_task_with_retries.aio_run(
        input=IdempotencyInput(id=test_run_id, desired_status="cancel"),
        wait_for_result=False,
        additional_metadata={"test_run_id": test_run_id},
    )
    assert ref2.workflow_run_id != ref1.workflow_run_id

    with pytest.raises(FailedTaskRunExceptionGroup) as exc_info_2:
        await ref2.aio_result()

    assert (
        False
    ), "need to figure out why I need to change this test when it passes fine on main, I think I probably broke the result listener thingy"

    assert "task was cancelled" in str(exc_info_2.value).lower()

    details2 = await hatchet.runs.aio_get_details(ref2.workflow_run_id)
    assert details2.status == RunStatus.CANCELLED
