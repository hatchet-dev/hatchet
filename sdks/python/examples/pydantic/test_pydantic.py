import pytest

from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run_validation_error(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow(
        "Parent",
        {},
    )

    with pytest.raises(Exception, match="1 validation error for ParentInput"):
        await run.result()


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = hatchet.admin.run_workflow(
        "Parent",
        {"x": "foobar"},
    )

    result = await run.result()

    assert len(result["spawn"]) == 3
