import asyncio

from hatchet_sdk import IdempotencyCollisionError

from examples.idempotency.worker import idempotent_task, IdempotencyInput


async def main() -> None:
    # > trigger
    ref_1 = await idempotent_task.aio_run(
        input=IdempotencyInput(id="123"),
        wait_for_result=False,
    )

    try:
        ref_2 = await idempotent_task.aio_run(
            input=IdempotencyInput(id="123"),
            wait_for_result=False,
        )
        run_id_2 = ref_2.workflow_run_id
    except IdempotencyCollisionError as e:
        print(
            f"Run with external ID {e.existing_run_external_id} already exists for this idempotency key"
        )
        run_id_2 = e.existing_run_external_id

    res_1 = await ref_1.aio_result()
    res_2 = await idempotent_task.aio_get_result(run_id_2)

    assert res_1 == res_2
    assert ref_1.workflow_run_id == run_id_2
    # !!
