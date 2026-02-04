import pytest

from examples.unit_testing.workflows import (Lifespan, UnitTestInput,
                                             UnitTestOutput,
                                             async_complex_workflow,
                                             async_simple_workflow,
                                             async_standalone,
                                             durable_async_complex_workflow,
                                             durable_async_simple_workflow,
                                             durable_async_standalone,
                                             durable_sync_complex_workflow,
                                             durable_sync_simple_workflow,
                                             durable_sync_standalone, start,
                                             sync_complex_workflow,
                                             sync_simple_workflow,
                                             sync_standalone)
from hatchet_sdk import Task


@pytest.mark.parametrize(
    "func",
    [
        sync_standalone,
        durable_sync_standalone,
        sync_simple_workflow,
        durable_sync_simple_workflow,
        sync_complex_workflow,
        durable_sync_complex_workflow,
    ],
)
def test_simple_unit_sync(func: Task[UnitTestInput, UnitTestOutput]) -> None:
    input = UnitTestInput(key="test_key", number=42)
    additional_metadata = {"meta_key": "meta_value"}
    lifespan = Lifespan(mock_db_url="sqlite:///:memory:")
    retry_count = 1

    expected_output = UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=additional_metadata,
        retry_count=retry_count,
        mock_db_url=lifespan.mock_db_url,
    )

    assert (
        func.mock_run(
            input=input,
            additional_metadata=additional_metadata,
            lifespan=lifespan,
            retry_count=retry_count,
            parent_outputs={start.name: expected_output.model_dump()},
        )
        == expected_output
    )


@pytest.mark.parametrize(
    "func",
    [
        async_standalone,
        durable_async_standalone,
        async_simple_workflow,
        durable_async_simple_workflow,
        async_complex_workflow,
        durable_async_complex_workflow,
    ],
)
@pytest.mark.asyncio(loop_scope="session")
async def test_simple_unit_async(func: Task[UnitTestInput, UnitTestOutput]) -> None:
    input = UnitTestInput(key="test_key", number=42)
    additional_metadata = {"meta_key": "meta_value"}
    lifespan = Lifespan(mock_db_url="sqlite:///:memory:")
    retry_count = 1

    expected_output = UnitTestOutput(
        key=input.key,
        number=input.number,
        additional_metadata=additional_metadata,
        retry_count=retry_count,
        mock_db_url=lifespan.mock_db_url,
    )

    assert (
        await func.aio_mock_run(
            input=input,
            additional_metadata=additional_metadata,
            lifespan=lifespan,
            retry_count=retry_count,
            parent_outputs={start.name: expected_output.model_dump()},
        )
        == expected_output
    )
