import pytest

from examples.simple.worker import simple


@pytest.mark.asyncio(loop_scope="session")
async def test_simple_workflow_running_options() -> None:
    x1 = simple.run()
    x2 = await simple.aio_run()

    x3 = simple.run_many([simple.create_bulk_run_item()])[0]
    x4 = (await simple.aio_run_many([simple.create_bulk_run_item()]))[0]

    x5 = simple.run_no_wait().result()
    x6 = (await simple.aio_run_no_wait()).result()
    x7 = [x.result() for x in simple.run_many_no_wait([simple.create_bulk_run_item()])][
        0
    ]
    x8 = [
        x.result()
        for x in await simple.aio_run_many_no_wait([simple.create_bulk_run_item()])
    ][0]

    x9 = await simple.run_no_wait().aio_result()
    x10 = await (await simple.aio_run_no_wait()).aio_result()
    x11 = [
        await x.aio_result()
        for x in simple.run_many_no_wait([simple.create_bulk_run_item()])
    ][0]
    x12 = [
        await x.aio_result()
        for x in await simple.aio_run_many_no_wait([simple.create_bulk_run_item()])
    ][0]

    assert all(
        x == {"result": "Hello, world!"}
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
