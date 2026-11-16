import pytest

from examples.bug_tests.dag_cancel_if_missing_output_key.worker import (
    cancel_if_missing_output_key_dag,
)


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_cancel_if_on_missing_output_key_does_not_fail_dag() -> None:
    result = await cancel_if_missing_output_key_dag.aio_run()

    assert result["parent_without_flag_key"] == {"value": 1}
    assert result["child_with_cancel_condition"] == {"ran": True}
