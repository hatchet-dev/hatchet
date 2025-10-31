import pytest

from examples.dataclasses.worker import say_hello, Input, Output
from hatchet_sdk.runnables.workflow import Standalone


@pytest.mark.parametrize("task", [say_hello])
@pytest.mark.asyncio(loop_scope="session")
async def test_dataclass_usage(
    task: Standalone[Input, Output],
) -> None:
    input = Input(name="Hatchet")
    x1 = task.run(input)
    x2 = await task.aio_run(input)

    x3 = task.run_many([task.create_bulk_run_item(input)])[0]
    x4 = (await task.aio_run_many([task.create_bulk_run_item(input)]))[0]

    x5 = task.run_no_wait(input).result()
    x6 = (await task.aio_run_no_wait(input)).result()
    x7 = [
        x.result() for x in task.run_many_no_wait([task.create_bulk_run_item(input)])
    ][0]
    x8 = [
        x.result()
        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item(input)])
    ][0]

    x9 = await task.run_no_wait(input).aio_result()
    x10 = await (await task.aio_run_no_wait(input)).aio_result()
    x11 = [
        await x.aio_result()
        for x in task.run_many_no_wait([task.create_bulk_run_item(input)])
    ][0]
    x12 = [
        await x.aio_result()
        for x in await task.aio_run_many_no_wait([task.create_bulk_run_item(input)])
    ][0]

    assert all(
        x == Output(message="Hello, Hatchet!")
        for x in [x1, x2, x3, x4, x5, x6, x7, x8, x9, x10, x11, x12]
    )
