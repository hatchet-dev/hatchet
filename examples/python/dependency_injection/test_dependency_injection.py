import pytest

from examples.dependency_injection.worker import (
    ASYNC_DEPENDENCY_VALUE,
    SYNC_DEPENDENCY_VALUE,
    Output,
    async_task_with_dependencies,
    di_workflow,
    durable_async_task_with_dependencies,
    sync_task_with_dependencies,
)
from hatchet_sdk import EmptyModel
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


@pytest.mark.asyncio(loop_scope="session")
async def test_di_workflows() -> None:
    result = await di_workflow.aio_run()

    assert len(result) == 3

    for output in result.values():
        parsed = Output.model_validate(output)

        assert parsed.sync_dep == SYNC_DEPENDENCY_VALUE
        assert parsed.async_dep == ASYNC_DEPENDENCY_VALUE
