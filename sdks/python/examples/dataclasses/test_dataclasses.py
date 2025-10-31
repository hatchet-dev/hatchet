import pytest

from examples.dataclasses.worker import Input, Output, say_hello


@pytest.mark.asyncio(loop_scope="session")
async def test_dataclass_usage() -> None:
    input = Input(name="Hatchet")
    x1 = say_hello.run(input)
    x2 = await say_hello.aio_run(input)

    x3 = say_hello.run_many([say_hello.create_bulk_run_item(input)])[0]
    x4 = (await say_hello.aio_run_many([say_hello.create_bulk_run_item(input)]))[0]

    x5 = say_hello.run_no_wait(input).result()
    x6 = (await say_hello.aio_run_no_wait(input)).result()
    x7 = [
        x.result()
        for x in say_hello.run_many_no_wait([say_hello.create_bulk_run_item(input)])
    ][0]
    x8 = [
        x.result()
        for x in await say_hello.aio_run_many_no_wait(
            [say_hello.create_bulk_run_item(input)]
        )
    ][0]

    x9 = await say_hello.run_no_wait(input).aio_result()
    x10 = await (await say_hello.aio_run_no_wait(input)).aio_result()
    x11 = [
        await x.aio_result()
        for x in say_hello.run_many_no_wait([say_hello.create_bulk_run_item(input)])
    ][0]
    x12 = [
        await x.aio_result()
        for x in await say_hello.aio_run_many_no_wait(
            [say_hello.create_bulk_run_item(input)]
        )
    ][0]

    assert all(
        x == Output(message="Hello, Hatchet!")
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
