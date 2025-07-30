import pytest

from examples.dependency_injection.worker import (
    Output,
    async_dep,
    async_task_with_dependencies,
    di_workflow,
    durable_async_task_with_dependencies,
    durable_sync_task_with_dependencies,
    sync_dep,
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
        durable_sync_task_with_dependencies,
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_di_standalones(
    task: Standalone[EmptyModel, Output],
) -> None:
    result = await task.aio_run()

    assert isinstance(result, Output)
    assert result.sync_dep == sync_dep()
    assert result.async_dep == await async_dep()


@pytest.mark.asyncio(loop_scope="session")
async def test_di_workflows() -> None:
    result = await di_workflow.aio_run()

    assert len(result) == 4

    for output in result.values():
        parsed = Output.model_validate(output)

        assert parsed.sync_dep == sync_dep()
        assert parsed.async_dep == await async_dep()
