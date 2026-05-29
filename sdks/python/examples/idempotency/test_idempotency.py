import pytest

from examples.idempotency.worker import idempotent_task, IdempotencyInput


from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_idempotency_keys_prevent_duplicate_runs(hatchet: Hatchet) -> None:
    ref1 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"), wait_for_result=False
    )
    ref2 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"), wait_for_result=False
    )

    print(ref1, ref2)

    assert False
