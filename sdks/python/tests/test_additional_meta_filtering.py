import pytest
import asyncio

from hatchet_sdk import Hatchet
from uuid import uuid4

from examples.simple.worker import simple


@pytest.mark.asyncio(loop_scope="session")
async def test_additional_meta_and_or_filters(hatchet: Hatchet) -> None:
    m1 = str(uuid4())
    m2 = str(uuid4())

    ref1 = await simple.aio_run(
        additional_metadata={
            "foo": m1,
            "bar": m1,
        },
        wait_for_result=False,
    )

    ref2 = await simple.aio_run(
        additional_metadata={
            "foo": m1,
            "bar": m2,
        },
        wait_for_result=False,
    )

    ref3 = await simple.aio_run(
        additional_metadata={
            "foo": m2,
            "bar": m1,
        },
        wait_for_result=False,
    )

    await asyncio.gather(
        ref1.aio_result(),
        ref2.aio_result(),
        ref3.aio_result(),
    )
    await asyncio.sleep(3)

    with_or = await hatchet.runs.aio_list(
        additional_metadata={"foo": m1, "bar": m1}, additional_metadata_operator="OR"
    )

    assert len(with_or.rows) == 3

    with_and = await hatchet.runs.aio_list(
        additional_metadata={"foo": m1, "bar": m1}, additional_metadata_operator="AND"
    )

    assert len(with_and.rows) == 1
    assert with_and.rows[0].metadata.id == ref1.workflow_run_id

    with_and_2 = await hatchet.runs.aio_list(
        additional_metadata={"foo": m2, "bar": m1}, additional_metadata_operator="AND"
    )

    assert len(with_and_2.rows) == 1
    assert with_and_2.rows[0].metadata.id == ref3.workflow_run_id
