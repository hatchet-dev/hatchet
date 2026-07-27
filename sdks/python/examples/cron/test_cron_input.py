import pytest
from typing import cast

from hatchet_sdk import Hatchet

from examples.cron.cron_input import cron_input_example_send_greeting


async def test_cron_workflow_has_input_on_proto() -> None:
    proto = cron_input_example_send_greeting.to_proto()
    assert proto.HasField("cron_input")


@pytest.mark.asyncio(loop_scope="session")
async def test_cron_input_workflow_running_options(hatchet: Hatchet) -> None:
    with hatchet.cron.client() as client:
        cron = await hatchet.cron.aio_list(
            workflow_id=cron_input_example_send_greeting.id
        )

        assert cron.rows is not None
        assert len(cron.rows) == 1
        cron_id = cron.rows[0].metadata.id

        trigger_res = hatchet.cron._wa(client).workflow_cron_trigger(
            tenant=hatchet.tenant_id,
            cron_workflow=cron_id,
        )

        ref = hatchet.runs.get_run_ref(trigger_res.external_id)
        res = cast(
            dict[str, str],
            (await ref.aio_result()).get("cron_input_example_send_greeting"),
        )

        assert res == {"message": "Hello, Hatchet!"}

    await hatchet.workflows.aio_delete(cron_input_example_send_greeting.id)
