import pytest

from examples.serde.worker import generate_result, read_result, serde_workflow


@pytest.mark.asyncio(loop_scope="session")
async def test_custom_serde() -> None:
    result = await serde_workflow.aio_run()
    assert result[generate_result.name]["result"] != "my_result"
    assert result[read_result.name]["final_result"] == "my_result"
