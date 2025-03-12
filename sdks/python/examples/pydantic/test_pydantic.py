import pytest

from examples.pydantic.worker import ParentInput, child_workflow, parent_workflow
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run_validation_error(hatchet: Hatchet, worker: Worker) -> None:

    with pytest.raises(Exception, match="2 validation error for ChildInput"):
        run = child_workflow.run(input={})  # type: ignore[arg-type]
        await run.aio_result()


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    run = parent_workflow.run(
        ParentInput(x="foobar"),
    )

    result = await run.aio_result()

    assert len(result["spawn"]) == 3
