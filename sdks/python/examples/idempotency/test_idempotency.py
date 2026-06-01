import pytest

from examples.idempotency.worker import idempotent_task, IdempotencyInput

from hatchet_sdk import Hatchet
from hatchet_sdk.exceptions import IdempotencyCollisionError


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs(hatchet: Hatchet) -> None:
    ref1 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"), wait_for_result=False
    )

    with pytest.raises(IdempotencyCollisionError) as exc_info:
        await idempotent_task.aio_run(
            input=IdempotencyInput(id="123"), wait_for_result=False
        )

    assert exc_info.value.existing_run_external_id == ref1.workflow_run_id
