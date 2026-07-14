import pytest

from examples.cron.cron_input import CronInput, cron_input_workflow
from hatchet_sdk import Hatchet


@pytest.mark.asyncio(loop_scope="session")
async def test_cron_input_workflow_running_options(hatchet: Hatchet) -> None:
    input = CronInput(name="Hatchet")

    result = cron_input_workflow.run(input)
    aio_result = await cron_input_workflow.aio_run(input)

    for r in (result, aio_result):
        assert r["greet"] == {"message": "Hello, Hatchet!"}

    proto = cron_input_workflow.to_proto()
    assert proto.HasField("cron_input")
