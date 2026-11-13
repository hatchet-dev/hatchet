import pytest

from examples.dependency_injection.worker import (
    ASYNC_CM_DEPENDENCY_VALUE,
    ASYNC_DEPENDENCY_VALUE,
    CHAINED_ASYNC_CM_VALUE,
    CHAINED_CM_VALUE,
    ROLLBACK_ERROR_PREFIX,
    SYNC_CM_DEPENDENCY_VALUE,
    SYNC_DEPENDENCY_VALUE,
    TASK_ERROR_MESSAGE,
    Output,
    TransactionRolledBackError,
    async_task_with_dependencies,
    di_workflow,
    durable_async_task_with_dependencies,
    failing_task_with_async_transaction,
    failing_task_with_exception_swallowing_dependency,
    failing_task_with_sync_transaction,
    failing_task_with_transaction_entered_before_exception_swallowing_dependency,
    sync_task_with_dependencies,
    task_with_shared_dependency,
    task_with_type_aliases,
)
from hatchet_sdk import EmptyModel
from hatchet_sdk.exceptions import FailedTaskRunExceptionGroup
from hatchet_sdk.runnables.workflow import Standalone


@pytest.mark.parametrize(
    "task",
    [
        async_task_with_dependencies,
        sync_task_with_dependencies,
        durable_async_task_with_dependencies,
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_di_standalones(
    task: Standalone[EmptyModel, Output],
) -> None:
    result = await task.aio_run()

    assert isinstance(result, Output)
    assert result.sync_dep == SYNC_DEPENDENCY_VALUE
    assert result.async_dep == ASYNC_DEPENDENCY_VALUE
    assert (
        result.async_cm_dep == ASYNC_CM_DEPENDENCY_VALUE + "_" + ASYNC_DEPENDENCY_VALUE
    )
    assert result.sync_cm_dep == SYNC_CM_DEPENDENCY_VALUE + "_" + SYNC_DEPENDENCY_VALUE
    assert result.chained_dep == "chained_" + CHAINED_CM_VALUE
    assert result.chained_async_dep == "chained_" + CHAINED_ASYNC_CM_VALUE


@pytest.mark.asyncio(loop_scope="session")
async def test_di_workflows() -> None:
    result = await di_workflow.aio_run()

    assert len(result) == 3

    for output in result.values():
        parsed = Output.model_validate(output)

        assert parsed.sync_dep == SYNC_DEPENDENCY_VALUE
        assert parsed.async_dep == ASYNC_DEPENDENCY_VALUE
        assert (
            parsed.async_cm_dep
            == ASYNC_CM_DEPENDENCY_VALUE + "_" + ASYNC_DEPENDENCY_VALUE
        )
        assert (
            parsed.sync_cm_dep == SYNC_CM_DEPENDENCY_VALUE + "_" + SYNC_DEPENDENCY_VALUE
        )
        assert parsed.chained_dep == "chained_" + CHAINED_CM_VALUE
        assert parsed.chained_async_dep == "chained_" + CHAINED_ASYNC_CM_VALUE


@pytest.mark.asyncio(loop_scope="session")
async def test_type_aliases() -> None:
    result = await task_with_type_aliases.aio_run(EmptyModel())
    assert result
    for key in result:
        assert result[key] is True


@pytest.mark.asyncio(loop_scope="session")
async def test_shared_dependency_is_resolved_once_per_run() -> None:
    result = await task_with_shared_dependency.aio_run()

    assert result.direct == result.first_consumer == result.second_consumer


@pytest.mark.parametrize(
    "task",
    [
        failing_task_with_sync_transaction,
        failing_task_with_async_transaction,
        failing_task_with_transaction_entered_before_exception_swallowing_dependency,
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_context_manager_dependency_sees_task_exception(
    task: Standalone[EmptyModel, None],
) -> None:
    with pytest.raises(FailedTaskRunExceptionGroup) as exception_group:
        await task.aio_run()

    [task_error] = exception_group.value.exceptions

    assert task_error.exc_type == TransactionRolledBackError.__name__
    assert task_error.exc == f"{ROLLBACK_ERROR_PREFIX}: {TASK_ERROR_MESSAGE}"


@pytest.mark.asyncio(loop_scope="session")
async def test_task_still_fails_when_dependency_swallows_exception() -> None:
    with pytest.raises(FailedTaskRunExceptionGroup) as exception_group:
        await failing_task_with_exception_swallowing_dependency.aio_run()

    [task_error] = exception_group.value.exceptions

    assert task_error.exc_type == RuntimeError.__name__
    assert task_error.exc == TASK_ERROR_MESSAGE
