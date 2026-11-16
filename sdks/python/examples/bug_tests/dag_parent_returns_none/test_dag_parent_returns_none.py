import pytest

from examples.bug_tests.dag_parent_returns_none.worker import parent_returns_none_dag


@pytest.mark.timeout(30, func_only=True)
@pytest.mark.asyncio(loop_scope="session")
async def test_child_can_read_outputs_of_parents_that_returned_nothing() -> None:
    result = await parent_returns_none_dag.aio_run()

    assert result["child_reading_parent_outputs"] == {
        "none_parent_output": {},
        "empty_dict_parent_output": {},
    }
