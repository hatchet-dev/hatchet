import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'python',
  content:
    'import pytest\n\nfrom examples.simple.worker import simple, simple_durable\nfrom hatchet_sdk import EmptyModel\nfrom hatchet_sdk.runnables.workflow import Standalone\n\n\n@pytest.mark.parametrize("task", [simple, simple_durable])\n@pytest.mark.asyncio(loop_scope="session")\nasync def test_simple_workflow_running_options(\n    task: Standalone[EmptyModel, dict[str, str]],\n) -> None:\n    x1 = task.run()\n    x2 = await task.aio_run()\n\n    x3 = task.run_many([task.create_bulk_run_item()])[0]\n    x4 = (await task.aio_run_many([task.create_bulk_run_item()]))[0]\n\n    x5 = task.run_no_wait().result()\n    x6 = (await task.aio_run_no_wait()).result()\n    x7 = [x.result() for x in task.run_many_no_wait([task.create_bulk_run_item()])][0]\n    x8 = [\n        x.result()\n        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item()])\n    ][0]\n\n    x9 = await task.run_no_wait().aio_result()\n    x10 = await (await task.aio_run_no_wait()).aio_result()\n    x11 = [\n        await x.aio_result()\n        for x in task.run_many_no_wait([task.create_bulk_run_item()])\n    ][0]\n    x12 = [\n        await x.aio_result()\n        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item()])\n    ][0]\n\n    assert all(\n        x == {"result": "Hello, world!"}\n        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]\n    )\n',
  source: 'out/python/simple/test_simple_workflow.py',
  blocks: {},
  highlights: {},
};

export default snippet;
