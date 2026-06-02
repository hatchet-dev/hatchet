import pytest

from examples.idempotency.worker import idempotent_task, IdempotencyInput, EVENT_KEY

from hatchet_sdk import Hatchet, IdempotencyCollisionError
from hatchet_sdk.clients.rest.models.v1_task_summary_list import V1TaskSummaryList
from uuid import uuid4
from datetime import timedelta, datetime, timezone
import asyncio


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

    await asyncio.sleep(3)

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
