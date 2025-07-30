import pytest

from examples.dependency_injection.worker import (
    Output,
    async_dep,
    async_task_with_dependencies,
    sync_dep,
    sync_task_with_dependencies,
)
from hatchet_sdk import EmptyModel
from hatchet_sdk.runnables.workflow import Standalone


@pytest.mark.parametrize(
    "task", [async_task_with_dependencies, sync_task_with_dependencies]
)
@pytest.mark.asyncio(loop_scope="session")
async def test_simple_workflow_running_options(
    task: Standalone[EmptyModel, Output],
) -> None:
    result = task.run()

    assert isinstance(result, Output)
    assert result.sync_dep == sync_dep()
    assert result.async_dep == await async_dep()
