import pytest

from examples.pydantic.worker import ParentInput, child_workflow, parent_workflow
from hatchet_sdk import Hatchet, Worker


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run_validation_error(hatchet: Hatchet, worker: Worker) -> None:

    with pytest.raises(Exception, match="2 validation errors for ChildInput"):
        child_workflow.run(input={})  # type: ignore[arg-type]


# requires scope module or higher for shared event loop
@pytest.mark.asyncio(scope="session")
@pytest.mark.parametrize("worker", ["pydantic"], indirect=True)
async def test_run(hatchet: Hatchet, worker: Worker) -> None:
    result = parent_workflow.run(
        ParentInput(x="foobar"),
    )

    assert len(result["spawn"]) == 2
