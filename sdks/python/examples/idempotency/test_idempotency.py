import pytest

from examples.idempotency.worker import idempotent_task, IdempotencyInput

from hatchet_sdk import Hatchet
from hatchet_sdk.exceptions import DedupeViolationError


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs(hatchet: Hatchet) -> None:
    ref1 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"), wait_for_result=False
    )
    ref2 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"), wait_for_result=False
    )

    assert ref1 is not None
    assert ref2 is not None

    assert ref1.workflow_run_id == ref2.workflow_run_id

    result1 = await ref1.aio_result()
    result2 = await ref2.aio_result()

    assert result1 == result2
