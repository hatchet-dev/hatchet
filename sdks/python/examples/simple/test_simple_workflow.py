import pytest

from examples.simple.worker import step1


@pytest.mark.asyncio(loop_scope="session")
async def test_simple_workflow_running_options() -> None:
    x1 = step1.run()
    x2 = await step1.aio_run()

    x3 = step1.run_many([step1.create_bulk_run_item()])[0]
    x4 = (await step1.aio_run_many([step1.create_bulk_run_item()]))[0]

    x5 = step1.run_no_wait().result()
    x6 = (await step1.aio_run_no_wait()).result()
    x7 = [x.result() for x in step1.run_many_no_wait([step1.create_bulk_run_item()])][0]
    x8 = [
        x.result()
        for x in await step1.aio_run_many_no_wait([step1.create_bulk_run_item()])
    ][0]

    x9 = await step1.run_no_wait().aio_result()
    x10 = await (await step1.aio_run_no_wait()).aio_result()
    x11 = [
        await x.aio_result()
        for x in step1.run_many_no_wait([step1.create_bulk_run_item()])
    ][0]
    x12 = [
        await x.aio_result()
        for x in await step1.aio_run_many_no_wait([step1.create_bulk_run_item()])
    ][0]

    assert all(
        x == {"result": "Hello, world!"}
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
