import pytest

from examples.dict_input.worker import Output, say_hello_unsafely


@pytest.mark.asyncio(loop_scope="session")
async def test_dict_input() -> None:
    input = {"name": "Hatchet"}

    x1 = say_hello_unsafely.run(input)
    x2 = await say_hello_unsafely.aio_run(input)

    x3 = say_hello_unsafely.run_many([say_hello_unsafely.create_bulk_run_item(input)])[
        0
    ]
    x4 = (
        await say_hello_unsafely.aio_run_many(
            [say_hello_unsafely.create_bulk_run_item(input)]
        )
    )[0]

    x5 = say_hello_unsafely.run_no_wait(input).result()
    x6 = (await say_hello_unsafely.aio_run_no_wait(input)).result()
    x7 = [
        x.result()
        for x in say_hello_unsafely.run_many_no_wait(
            [say_hello_unsafely.create_bulk_run_item(input)]
        )
    ][0]
    x8 = [
        x.result()
        for x in await say_hello_unsafely.aio_run_many_no_wait(
            [say_hello_unsafely.create_bulk_run_item(input)]
        )
    ][0]

    x9 = await say_hello_unsafely.run_no_wait(input).aio_result()
    x10 = await (await say_hello_unsafely.aio_run_no_wait(input)).aio_result()
    x11 = [
        await x.aio_result()
        for x in say_hello_unsafely.run_many_no_wait(
            [say_hello_unsafely.create_bulk_run_item(input)]
        )
    ][0]
    x12 = [
        await x.aio_result()
        for x in await say_hello_unsafely.aio_run_many_no_wait(
            [say_hello_unsafely.create_bulk_run_item(input)]
        )
    ][0]

    assert all(
        x == Output(message="Hello, Hatchet!")
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
