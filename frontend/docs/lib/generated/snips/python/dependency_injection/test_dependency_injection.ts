import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "python",
  "content": "import pytest\n\nfrom examples.dependency_injection.worker import (\n    ASYNC_DEPENDENCY_VALUE,\n    SYNC_DEPENDENCY_VALUE,\n    Output,\n    async_dep,\n    async_task_with_dependencies,\n    di_workflow,\n    durable_async_task_with_dependencies,\n    durable_sync_task_with_dependencies,\n    sync_dep,\n    sync_task_with_dependencies,\n)\nfrom hatchet_sdk import EmptyModel\nfrom hatchet_sdk.runnables.workflow import Standalone\n\n\n@pytest.mark.parametrize(\n    \"task\",\n    [\n        async_task_with_dependencies,\n        sync_task_with_dependencies,\n        durable_async_task_with_dependencies,\n        durable_sync_task_with_dependencies,\n    ],\n)\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_di_standalones(\n    task: Standalone[EmptyModel, Output],\n) -> None:\n    result = await task.aio_run()\n\n    assert isinstance(result, Output)\n    assert result.sync_dep == SYNC_DEPENDENCY_VALUE\n    assert result.async_dep == ASYNC_DEPENDENCY_VALUE\n\n\n@pytest.mark.asyncio(loop_scope=\"session\")\nasync def test_di_workflows() -> None:\n    result = await di_workflow.aio_run()\n\n    assert len(result) == 4\n\n    for output in result.values():\n        parsed = Output.model_validate(output)\n\n        assert parsed.sync_dep == SYNC_DEPENDENCY_VALUE\n        assert parsed.async_dep == ASYNC_DEPENDENCY_VALUE\n",
  "source": "out/python/dependency_injection/test_dependency_injection.py",
  "blocks": {},
  "highlights": {}
};

export default snippet;
