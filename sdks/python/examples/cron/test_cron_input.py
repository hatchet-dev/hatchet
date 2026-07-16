import pytest

from examples.cron.cron_input import CronInput, cron_input_example_send_greeting


@pytest.mark.asyncio(loop_scope="session")
async def test_cron_input_workflow_running_options() -> None:
    input = CronInput(name="Hatchet")

    result = cron_input_example_send_greeting.run(input)
    aio_result = await cron_input_example_send_greeting.aio_run(input)

    for r in (result, aio_result):
        assert r == {"message": "Hello, Hatchet!"}

    proto = cron_input_example_send_greeting.to_proto()
    assert proto.HasField("cron_input")
