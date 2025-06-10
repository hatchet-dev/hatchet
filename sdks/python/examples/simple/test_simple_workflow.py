import pytest

from examples.simple.worker import simple, simple_durable
from hatchet_sdk import EmptyModel
from hatchet_sdk.runnables.workflow import Standalone


@pytest.mark.parametrize("task", [simple, simple_durable])
@pytest.mark.asyncio(loop_scope="session")
async def test_simple_workflow_running_options(
    task: Standalone[EmptyModel, dict[str, str]],
) -> None:
    x1 = task.run()
    x2 = await task.aio_run()

    x3 = task.run_many([task.create_bulk_run_item()])[0]
    x4 = (await task.aio_run_many([task.create_bulk_run_item()]))[0]

    x5 = task.run_no_wait().result()
    x6 = (await task.aio_run_no_wait()).result()
    x7 = [x.result() for x in task.run_many_no_wait([task.create_bulk_run_item()])][0]
    x8 = [
        x.result()
        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item()])
    ][0]

    x9 = await task.run_no_wait().aio_result()
    x10 = await (await task.aio_run_no_wait()).aio_result()
    x11 = [
        await x.aio_result()
        for x in task.run_many_no_wait([task.create_bulk_run_item()])
    ][0]
    x12 = [
        await x.aio_result()
        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item()])
    ][0]

    assert all(
        x == {"result": "Hello, world!"}
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
