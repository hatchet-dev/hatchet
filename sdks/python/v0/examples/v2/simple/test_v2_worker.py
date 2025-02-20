import pytest

from examples.v2.simple.worker import MyResultType, my_durable_func, my_func
from hatchet_sdk import Hatchet, Worker
from hatchet_sdk.workflow_run import RunRef


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["v2_simple"], indirect=True)
async def test_durable(hatchet: Hatchet, worker: Worker) -> None:
    durable_run: RunRef[dict[str, str]] = hatchet.admin.run(
        my_durable_func, {"test": "test"}
    )
    result = await durable_run.result()

    assert result == {"my_durable_func": "testing123"}


@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["v2_simple"], indirect=True)
async def test_func(hatchet: Hatchet, worker: Worker) -> None:
    durable_run: RunRef[MyResultType] = hatchet.admin.run(my_func, {"test": "test"})
    result = await durable_run.result()

    assert result == {"my_func": "testing123"}
