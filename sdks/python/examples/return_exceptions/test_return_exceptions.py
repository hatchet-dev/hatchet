import pytest

from examples.return_exceptions.worker import (
    Input,
    exception_parsing_workflow,
    return_exceptions_task,
)
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup


@pytest.mark.asyncio(loop_scope="session")
async def test_return_exceptions_async() -> None:
    results = await return_exceptions_task.aio_run_many(
        [
            return_exceptions_task.create_bulk_run_item(input=Input(index=i))
            for i in range(10)
        ],
        return_exceptions=True,
    )

    for i, result in enumerate(results):
        if i % 2 == 0:
            assert isinstance(result, Exception)
            assert f"error in task with index {i}" in str(result)
        else:
            assert result == {"message": "this is a successful task."}


def test_return_exceptions_sync() -> None:
    results = return_exceptions_task.run_many(
        [
            return_exceptions_task.create_bulk_run_item(input=Input(index=i))
            for i in range(10)
        ],
        return_exceptions=True,
    )

    for i, result in enumerate(results):
        if i % 2 == 0:
            assert isinstance(result, Exception)
            assert f"error in task with index {i}" in str(result)
        else:
            assert result == {"message": "this is a successful task."}


@pytest.mark.asyncio(loop_scope="session")
async def test_exceptions_parsing() -> None:
    with pytest.raises(FailedTaskRunExceptionGroup) as exc_info:
        await exception_parsing_workflow.aio_run()

    for exception in exc_info.value.exceptions:
        # Test that we don't get empty error messages
        assert exception.exc
        assert str(exception)
